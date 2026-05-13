package views

//go:generate mockgen -source=interface.go -destination=mock_repository.go -package=views

// IRepository defines the read operations available for database views declared
// under db/views.
type IRepository interface {
	// Uses db/views/vw_cmk_key.sql
	ReadCmkKeyView() ([]CmkKeyView, error)
	QueryCmkKeyView(query string, args ...any) ([]CmkKeyView, error)

	// Uses db/views/vw_cmk_creation_key_queue.sql
	ReadCmkCreationKeyQueueView() ([]CmkCreationKeyQueueView, error)
	QueryCmkCreationKeyQueueView(query string, args ...any) ([]CmkCreationKeyQueueView, error)
	CountCmkCreationKeyQueueView(query string, args ...any) (uint, error)

	// Backward-compatible aliases for the previous key_creation naming.
	ReadCmkKeyCreationQueueView() ([]CmkKeyCreationQueueView, error)
	QueryCmkKeyCreationQueueView(query string, args ...any) ([]CmkKeyCreationQueueView, error)

	// Uses db/views/vw_cmk_key_version.sql
	ReadCmkKeyVersionView() ([]CmkKeyVersionView, error)
	QueryCmkKeyVersionView(query string, args ...any) ([]CmkKeyVersionView, error)

	// Uses db/views/vw_cmk_wrapping_key_ref.sql
	ReadCmkWrappingKeyRefView() ([]CmkWrappingKeyRefView, error)
	QueryCmkWrappingKeyRefView(query string, args ...any) ([]CmkWrappingKeyRefView, error)
}
