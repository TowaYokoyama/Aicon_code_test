package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"Aicon-assignment/internal/domain/entity"
	domainErrors "Aicon-assignment/internal/domain/errors"
	"Aicon-assignment/internal/usecase"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockItemUsecase struct {
	mock.Mock
}

func (m *MockItemUsecase) GetAllItems(ctx context.Context) ([]*entity.Item, error) {
	args := m.Called(ctx)
	if items, ok := args.Get(0).([]*entity.Item); ok {
		return items, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockItemUsecase) GetItemByID(ctx context.Context, id int64) (*entity.Item, error) {
	args := m.Called(ctx, id)
	if item, ok := args.Get(0).(*entity.Item); ok {
		return item, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockItemUsecase) CreateItem(ctx context.Context, input usecase.CreateItemInput) (*entity.Item, error) {
	args := m.Called(ctx, input)
	if item, ok := args.Get(0).(*entity.Item); ok {
		return item, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockItemUsecase) DeleteItem(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockItemUsecase) GetCategorySummary(ctx context.Context) (*usecase.CategorySummary, error) {
	args := m.Called(ctx)
	if summary, ok := args.Get(0).(*usecase.CategorySummary); ok {
		return summary, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockItemUsecase) PatchItem(ctx context.Context, id int64, input usecase.PatchItemInput) (*entity.Item, error) {
	args := m.Called(ctx, id, input)
	if item, ok := args.Get(0).(*entity.Item); ok {
		return item, args.Error(1)
	}
	return nil, args.Error(1)
}

func newPatchContext(t *testing.T, body string, idParam string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/items/"+idParam, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(idParam)
	return c, rec
}

func TestItemHandler_PatchItem(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	updatedItem := &entity.Item{
		ID:            1,
		Name:          "Updated",
		Category:      "時計",
		Brand:         "ROLEX",
		PurchasePrice: 1000,
		PurchaseDate:  "2023-01-01",
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now,
	}

	tests := []struct {
		name         string
		idParam      string
		body         string
		setupMock    func(*MockItemUsecase)
		wantStatus   int
		wantError    *ErrorResponse
		wantItem     *entity.Item
		assertNoCall bool
	}{
		{
			name:    "invalid id",
			idParam: "abc",
			body:    `{"name":"Updated"}`,
			setupMock: func(_ *MockItemUsecase) {
			},
			wantStatus:   http.StatusBadRequest,
			wantError:    &ErrorResponse{Error: "invalid item ID"},
			assertNoCall: true,
		},
		{
			name:    "invalid json",
			idParam: "1",
			body:    `{`,
			setupMock: func(_ *MockItemUsecase) {
			},
			wantStatus:   http.StatusBadRequest,
			wantError:    &ErrorResponse{Error: "invalid request format"},
			assertNoCall: true,
		},
		{
			name:    "not found",
			idParam: "1",
			body:    `{"name":"Updated"}`,
			setupMock: func(m *MockItemUsecase) {
				m.On("PatchItem", mock.Anything, int64(1), mock.MatchedBy(func(in usecase.PatchItemInput) bool {
					return in.Name != nil && *in.Name == "Updated"
				})).Return((*entity.Item)(nil), domainErrors.ErrItemNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantError:  &ErrorResponse{Error: "item not found"},
		},
		{
			name:    "validation error",
			idParam: "1",
			body:    `{"name":""}`,
			setupMock: func(m *MockItemUsecase) {
				m.On("PatchItem", mock.Anything, int64(1), mock.Anything).Return((*entity.Item)(nil), domainErrors.ErrInvalidInput)
			},
			wantStatus: http.StatusBadRequest,
			wantError:  &ErrorResponse{Error: "validation failed", Details: []string{domainErrors.ErrInvalidInput.Error()}},
		},
		{
			name:    "success",
			idParam: "1",
			body:    `{"name":"Updated"}`,
			setupMock: func(m *MockItemUsecase) {
				m.On("PatchItem", mock.Anything, int64(1), mock.MatchedBy(func(in usecase.PatchItemInput) bool {
					return in.Name != nil && *in.Name == "Updated"
				})).Return(updatedItem, nil)
			},
			wantStatus: http.StatusOK,
			wantItem:   updatedItem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsecase := new(MockItemUsecase)
			tt.setupMock(mockUsecase)

			handler := NewItemHandler(mockUsecase)
			c, rec := newPatchContext(t, tt.body, tt.idParam)

			err := handler.PatchItem(c)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.assertNoCall {
				mockUsecase.AssertNotCalled(t, "PatchItem", mock.Anything, mock.Anything, mock.Anything)
			}

			if tt.wantError != nil {
				var got ErrorResponse
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Equal(t, tt.wantError.Error, got.Error)
				assert.Equal(t, tt.wantError.Details, got.Details)
			}

			if tt.wantItem != nil {
				var got entity.Item
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Equal(t, tt.wantItem.ID, got.ID)
				assert.Equal(t, tt.wantItem.Name, got.Name)
				assert.Equal(t, tt.wantItem.Brand, got.Brand)
				assert.Equal(t, tt.wantItem.Category, got.Category)
				assert.Equal(t, tt.wantItem.PurchasePrice, got.PurchasePrice)
				assert.Equal(t, tt.wantItem.PurchaseDate, got.PurchaseDate)
			}

			mockUsecase.AssertExpectations(t)
		})
	}
}
