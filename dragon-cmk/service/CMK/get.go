package cmk

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/PointerByte/lock-max/dragon-cmk/service/utilities"
	"github.com/google/uuid"
)

func getHealthz(status commonEntity.KeyStatus) string {
	switch status {
	case commonEntity.KeyStatusPendingImport:
		return "PENDING_IMPORT"
	case commonEntity.KeyStatusEnabled:
		return "ok"
	case commonEntity.KeyStatusDisabled:
		return "DISABLED"
	case commonEntity.KeyStatusPendingDeletion:
		return "PENDING_DELETION"
	case commonEntity.KeyStatusUnavailable:
		return "UNAVAILABLE"
	default:
		return "UNKNOWN"
	}
}

func (r *Repository) Status() (status *models.StatusResponse, _ error) {
	kekData, err := r.kek.GetKEK(uuid.Nil, "")
	if err != nil {
		return nil, err
	}

	idCmkKey, _, err := utilities.GetSecretCmkKey(kekData.SecretCmkKey)
	if err != nil {
		return nil, err
	}

	cmkKey, err := r.getKey(*idCmkKey)
	if err != nil {
		return nil, err
	}

	status = &models.StatusResponse{}
	status.ID = cmkKey.IDCmkKey
	status.Healthz = getHealthz(cmkKey.Status)
	version, err := r.getKeyVersion(cmkKey.IDCmkKeyVersion)
	if err != nil {
		return nil, err
	}
	status.Version = fmt.Sprintf("v%d", version.VersionNumber)
	return status, nil
}

func validatePagination(page uint, totalRegisterPage uint) error {
	if page == 0 {
		return errors.New("page must be greater than zero")
	}
	if totalRegisterPage == 0 {
		return errors.New("totalResgisterPage must be greater than zero")
	}
	return nil
}

func newPagination(totalRegisters uint, page uint, totalRegisterPage uint) models.Pagination {
	totalPages := totalRegisters / totalRegisterPage
	if totalRegisters%totalRegisterPage != 0 {
		totalPages++
	}
	return models.Pagination{
		TotalRegisters:     totalRegisters,
		TotalPages:         totalPages,
		TotalRegistersPage: totalRegisterPage,
		PageNow:            page,
	}
}

func paginateUUIDs(values []uuid.UUID, page uint, totalRegisterPage uint) []uuid.UUID {
	start := int((page - 1) * totalRegisterPage)
	if start >= len(values) {
		return nil
	}
	end := start + int(totalRegisterPage)
	if end > len(values) {
		end = len(values)
	}
	return values[start:end]
}

func (r *Repository) ListCmkKey(idKek uuid.UUID, page uint, totalRegisterPage uint) (*models.PaginatedCmkKey, error) {
	if err := validatePagination(page, totalRegisterPage); err != nil {
		return nil, err
	}

	versions, err := r.views.QueryCmkKeyVersionView("WHERE id_cmk_wrapping_key_ref = $1", idKek)
	if err != nil {
		return nil, err
	}

	seen := make(map[uuid.UUID]struct{}, len(versions))
	idsCmkKey := make([]uuid.UUID, 0, len(versions))
	for _, version := range versions {
		if _, ok := seen[version.IDCmkKey]; ok {
			continue
		}
		seen[version.IDCmkKey] = struct{}{}
		idsCmkKey = append(idsCmkKey, version.IDCmkKey)
	}
	sort.Slice(idsCmkKey, func(i, j int) bool {
		return idsCmkKey[i].String() < idsCmkKey[j].String()
	})

	results := make([]models.CmkKey, 0, totalRegisterPage)
	for _, idCmkKey := range paginateUUIDs(idsCmkKey, page, totalRegisterPage) {
		cmkKey, err := r.views.QueryCmkKeyView("WHERE id_cmk_key = $1", idCmkKey)
		if err != nil {
			return nil, err
		}
		if len(cmkKey) == 0 {
			return nil, fmt.Errorf("the key %s was not found", idCmkKey.String())
		}
		k := cmkKey[0]
		data, err := r.views.QueryCmkCreationKeyQueueView("WHERE id_cmk_key = $1", k.IDCmkKey)
		if err != nil {
			return nil, err
		}

		if len(data) == 0 {
			return nil, fmt.Errorf("the key %s no have relation with queue %s", k.IDCmkKey.String(), k.IDCmkKey.String())
		}

		results = append(results, models.CmkKey{
			CmkKey: k,
			Queue:  data[0],
		})
	}

	return &models.PaginatedCmkKey{
		Results:    results,
		Pagination: newPagination(uint(len(idsCmkKey)), page, totalRegisterPage),
	}, nil
}

func (r *Repository) ListCreationKeyQueues(idCmkKey uuid.UUID, status commonEntity.QueueStatus, page uint, totalRegisterPage uint) (*models.PaginatedCreationKeyQueue, error) {
	if err := validatePagination(page, totalRegisterPage); err != nil {
		return nil, err
	}

	whereClause, args := creationKeyQueueListWhereClause(idCmkKey, status)
	totalRegisters, err := r.views.CountCmkCreationKeyQueueView(whereClause, args...)
	if err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, int(totalRegisterPage), int((page-1)*totalRegisterPage))
	queryClause := creationKeyQueueListQueryClause(whereClause, len(args)+1)

	results, err := r.views.QueryCmkCreationKeyQueueView(queryClause, queryArgs...)
	if err != nil {
		return nil, err
	}

	return &models.PaginatedCreationKeyQueue{
		Results:    results,
		Pagination: newPagination(totalRegisters, page, totalRegisterPage),
	}, nil
}

func creationKeyQueueListWhereClause(idCmkKey uuid.UUID, status commonEntity.QueueStatus) (string, []any) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if idCmkKey != uuid.Nil {
		args = append(args, idCmkKey)
		clauses = append(clauses, fmt.Sprintf("id_cmk_key = $%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func creationKeyQueueListQueryClause(whereClause string, nextArgIndex int) string {
	limitArg := nextArgIndex
	offsetArg := nextArgIndex + 1
	if whereClause == "" {
		return fmt.Sprintf("ORDER BY queued_at DESC, id_cmk_key_creation_queue DESC LIMIT $%d OFFSET $%d", limitArg, offsetArg)
	}
	return fmt.Sprintf("%s ORDER BY queued_at DESC, id_cmk_key_creation_queue DESC LIMIT $%d OFFSET $%d", whereClause, limitArg, offsetArg)
}
