package diego_errors

func SanitizeErrorMessage(message string) string {
	switch message {
	case
		INSUFFICIENT_RESOURCES_MESSAGE,
		MISSING_APP_BITS_DOWNLOAD_URI_MESSAGE,
		MISSING_APP_ID_MESSAGE,
		MISSING_TASK_ID_MESSAGE,
		NO_COMPILER_DEFINED_MESSAGE,
		CELL_MISMATCH_MESSAGE:
		return message
	default:
		return "staging failed"
	}
}
