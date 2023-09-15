package main

import "log"

// NugetServerCredentials are credentials defined in the CI server and injected into this trusted image
type NugetServerCredentials struct {
	Name                 string                                     `json:"name,omitempty"`
	Type                 string                                     `json:"type,omitempty"`
	AdditionalProperties NugetServerCredentialsAdditionalProperties `json:"additionalProperties,omitempty"`
}

// NugetServerCredentialsAdditionalProperties has additional properties for the Nuget Server credentials
type NugetServerCredentialsAdditionalProperties struct {
	APIURL string `json:"apiUrl,omitempty"`
	APIKey string `json:"apiKey,omitempty"`
}

// SonarQubeServerCredentials are credentials defined in the CI server and injected into this trusted image
type SonarQubeServerCredentials struct {
	Name                 string                                         `json:"name,omitempty"`
	Type                 string                                         `json:"type,omitempty"`
	AdditionalProperties SonarQubeServerCredentialsAdditionalProperties `json:"additionalProperties,omitempty"`
}

// SonarQubeServerCredentialsAdditionalProperties has additional properties for the SonarQube Server credentials
type SonarQubeServerCredentialsAdditionalProperties struct {
	APIURL string `json:"apiUrl,omitempty"`
	Token  string `json:"token,omitempty"`
}

// GetNugetServerCredentialsByName returns a credential with the specified name
func GetNugetServerCredentialsByName(c []NugetServerCredentials, name string) *NugetServerCredentials {

	log.Printf("Looking for credential with name %v...", name)
	for _, cred := range c {
		log.Printf("Checking credential %v...", cred.Name)
		if cred.Name == name {
			log.Printf("Credential with name %v was retrieved.", name)
			return &cred
		}
	}

	log.Printf("Credential with name %v was not found.", name)
	return nil
}

// GetSonarQubeServerCredentialsByName returns a credential with the specified name
func GetSonarQubeServerCredentialsByName(c []SonarQubeServerCredentials, name string) *SonarQubeServerCredentials {

	log.Printf("Looking for credential with name %v...", name)
	for _, cred := range c {
		log.Printf("Checking credential %v...", cred.Name)
		if cred.Name == name {
			log.Printf("Credential with name %v was retrieved.", name)
			return &cred
		}
	}

	log.Printf("Credential with name %v was not found.", name)
	return nil
}
