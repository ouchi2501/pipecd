// Copyright 2021 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sourcedecrypter

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pipe-cd/pipecd/pkg/config"
)

type secretDecrypter interface {
	Decrypt(string) (string, error)
}

func DecryptSecrets(appDir string, enc config.SecretEncryption, dcr secretDecrypter) error {
	if len(enc.DecryptionTargets) == 0 {
		return nil
	}
	if len(enc.EncryptedSecrets) == 0 {
		return fmt.Errorf("no encrypted secret was specified to decrypt (%q)", enc.DecryptionTargets)
	}

	secrets := make(map[string]string, len(enc.EncryptedSecrets))
	for k, v := range enc.EncryptedSecrets {
		ds, err := dcr.Decrypt(v)
		if err != nil {
			return fmt.Errorf("failed to decrypt %s secret (%w)", k, err)
		}
		secrets[k] = ds
	}
	data := map[string](map[string]string){
		"encryptedSecrets": secrets,
	}

	for _, t := range enc.DecryptionTargets {
		targetPath := filepath.Join(appDir, t)
		tmpl, err := template.ParseFiles(targetPath)
		if err != nil {
			return fmt.Errorf("failed to parse decryption target %s (%w)", t, err)
		}

		// Return an error immediately if the target is using a nonexistent secret.
		tmpl = tmpl.Option("missingkey=error")

		f, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to open decryption target %s (%w)", t, err)
		}

		if err := tmpl.Execute(f, data); err != nil {
			f.Close()
			return fmt.Errorf("failed to render decryption target %s (%w)", t, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close decryption target %s (%w)", t, err)
		}
	}

	return nil
}
