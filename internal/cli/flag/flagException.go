package flag

import "flag"

func FlagException(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case flag.ErrHelp:
		return nil // No action needed, help message is displayed
	default:
		return err // Return the error for further handling
	}
}
