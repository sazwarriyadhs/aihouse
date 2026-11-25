package worker

// ValidateJob exposes the existing queue payload validation to specialized
// handlers without changing the validation rules.
func ValidateJob(jobType string, payload string) error {
	return validateJob(jobType, payload)
}
