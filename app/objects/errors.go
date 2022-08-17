package objects

func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "record not found"
}
