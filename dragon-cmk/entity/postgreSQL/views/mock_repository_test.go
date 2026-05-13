package views

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

func TestMockRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	mockErr := errors.New("mock error")

	expectedKeyView := []CmkKeyView{{IDCmkKey: id}}
	expectedQueueView := []CmkCreationKeyQueueView{{IDCmkKeyCreationQueue: id}}
	expectedVersionView := []CmkKeyVersionView{{IDCmkKeyVersion: id, KID: "kid-1"}}
	expectedWrappingView := []CmkWrappingKeyRefView{{IDCmkWrappingKeyRef: id, Provider: "aws", Version: "v1"}}

	repo.EXPECT().ReadCmkKeyView().Return(expectedKeyView, nil)
	repo.EXPECT().ReadCmkCreationKeyQueueView().Return(expectedQueueView, nil)
	repo.EXPECT().ReadCmkKeyVersionView().Return(expectedVersionView, nil)
	repo.EXPECT().QueryCmkKeyView("WHERE id_cmk_key = $1", id).Return(expectedKeyView, nil)
	repo.EXPECT().QueryCmkCreationKeyQueueView("WHERE status = $1", "pending").Return(expectedQueueView, nil)
	repo.EXPECT().QueryCmkKeyVersionView("WHERE kid = $1", "kid-1").Return(expectedVersionView, nil)
	repo.EXPECT().QueryCmkWrappingKeyRefView("WHERE provider = $1", "aws").Return(expectedWrappingView, nil)
	repo.EXPECT().ReadCmkWrappingKeyRefView().Return(expectedWrappingView, nil)
	repo.EXPECT().ReadCmkWrappingKeyRefView().Return(nil, mockErr)

	if got, err := repo.ReadCmkKeyView(); err != nil || len(got) != 1 {
		t.Fatalf("ReadCmkKeyView() = (%v, %v)", got, err)
	}

	if got, err := repo.ReadCmkCreationKeyQueueView(); err != nil || len(got) != 1 {
		t.Fatalf("ReadCmkCreationKeyQueueView() = (%v, %v)", got, err)
	}

	if got, err := repo.ReadCmkKeyVersionView(); err != nil || len(got) != 1 {
		t.Fatalf("ReadCmkKeyVersionView() = (%v, %v)", got, err)
	}

	if got, err := repo.QueryCmkKeyView("WHERE id_cmk_key = $1", id); err != nil || len(got) != 1 {
		t.Fatalf("QueryCmkKeyView() = (%v, %v)", got, err)
	}

	if got, err := repo.QueryCmkCreationKeyQueueView("WHERE status = $1", "pending"); err != nil || len(got) != 1 {
		t.Fatalf("QueryCmkCreationKeyQueueView() = (%v, %v)", got, err)
	}

	if got, err := repo.QueryCmkKeyVersionView("WHERE kid = $1", "kid-1"); err != nil || len(got) != 1 {
		t.Fatalf("QueryCmkKeyVersionView() = (%v, %v)", got, err)
	}

	if got, err := repo.QueryCmkWrappingKeyRefView("WHERE provider = $1", "aws"); err != nil || len(got) != 1 {
		t.Fatalf("QueryCmkWrappingKeyRefView() = (%v, %v)", got, err)
	}

	if got, err := repo.ReadCmkWrappingKeyRefView(); err != nil || len(got) != 1 {
		t.Fatalf("ReadCmkWrappingKeyRefView() = (%v, %v)", got, err)
	}

	if got, err := repo.ReadCmkWrappingKeyRefView(); !errors.Is(err, mockErr) || got != nil {
		t.Fatalf("ReadCmkWrappingKeyRefView() = (%v, %v)", got, err)
	}
}
