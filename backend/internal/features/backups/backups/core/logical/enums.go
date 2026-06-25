package backups_core_logical

type BackupStatus string

const (
	BackupStatusInProgress BackupStatus = "IN_PROGRESS"
	BackupStatusCompleted  BackupStatus = "COMPLETED"
	BackupStatusFailed     BackupStatus = "FAILED"
	BackupStatusCanceled   BackupStatus = "CANCELED"
)

type RestoreVerificationStatus string

const (
	RestoreVerificationStatusNotVerified        RestoreVerificationStatus = "NOT_VERIFIED"
	RestoreVerificationStatusVerifiedSuccessful RestoreVerificationStatus = "VERIFIED_SUCCESSFUL"
	RestoreVerificationStatusVerificationFailed RestoreVerificationStatus = "VERIFICATION_FAILED"
)
