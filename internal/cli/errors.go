package cli

func formatErr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
