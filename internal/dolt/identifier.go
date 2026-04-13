package dolt

import "regexp"

var databaseIdentifierRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

func validateDatabaseIdentifier(name string) error {
	if !databaseIdentifierRe.MatchString(name) {
		return &InvalidDatabaseIdentifierError{Name: name}
	}
	return nil
}
