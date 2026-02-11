package usecase

import (
	"context"
	"fmt"
	"time"

	"Aicon-assignment/internal/domain/entity"
	domainErrors "Aicon-assignment/internal/domain/errors"
)

type ItemUsecase interface {
	GetAllItems(ctx context.Context) ([]*entity.Item, error)
	GetItemByID(ctx context.Context, id int64) (*entity.Item, error)
	CreateItem(ctx context.Context, input CreateItemInput) (*entity.Item, error)
	DeleteItem(ctx context.Context, id int64) error
	GetCategorySummary(ctx context.Context) (*CategorySummary, error)
	PatchItem(ctx context.Context, id int64, input PatchItemInput) (*entity.Item, error)
}

type CreateItemInput struct {
	Name          string `json:"name"`
	Category      string `json:"category"`
	Brand         string `json:"brand"`
	PurchasePrice int    `json:"purchase_price"`
	PurchaseDate  string `json:"purchase_date"`
}

type CategorySummary struct {
	Categories map[string]int `json:"categories"`
	Total      int            `json:"total"`
}

type itemUsecase struct {
	itemRepo ItemRepository
}
type PatchItemInput struct {
	Name          *string `json:"name,omitempty"`
	Brand         *string `json:"brand,omitempty"`
	PurchasePrice *int    `json:"purchase_price,omitempty"`
}

func NewItemUsecase(itemRepo ItemRepository) ItemUsecase {
	return &itemUsecase{
		itemRepo: itemRepo,
	}
}

func (u *itemUsecase) GetAllItems(ctx context.Context) ([]*entity.Item, error) {
	items, err := u.itemRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve items: %w", err)
	}

	return items, nil
}

func (u *itemUsecase) GetItemByID(ctx context.Context, id int64) (*entity.Item, error) {
	if id <= 0 {
		return nil, domainErrors.ErrInvalidInput
	}

	item, err := u.itemRepo.FindByID(ctx, id)
	if err != nil {
		if domainErrors.IsNotFoundError(err) {
			return nil, domainErrors.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to retrieve item: %w", err)
	}

	return item, nil
}

func (u *itemUsecase) CreateItem(ctx context.Context, input CreateItemInput) (*entity.Item, error) {
	// バリデーションして、新しいエンティティを作成
	item, err := entity.NewItem(
		input.Name,
		input.Category,
		input.Brand,
		input.PurchasePrice,
		input.PurchaseDate,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrInvalidInput, err.Error())
	}

	createdItem, err := u.itemRepo.Create(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	return createdItem, nil
}

func (u *itemUsecase) DeleteItem(ctx context.Context, id int64) error {
	if id <= 0 {
		return domainErrors.ErrInvalidInput
	}

	_, err := u.itemRepo.FindByID(ctx, id)
	if err != nil {
		if domainErrors.IsNotFoundError(err) {
			return domainErrors.ErrItemNotFound
		}
		return fmt.Errorf("failed to check item existence: %w", err)
	}

	err = u.itemRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}

func (u *itemUsecase) GetCategorySummary(ctx context.Context) (*CategorySummary, error) {
	categoryCounts, err := u.itemRepo.GetSummaryByCategory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get category summary: %w", err)
	}

	// 合計計算
	total := 0
	for _, count := range categoryCounts {
		total += count
	}

	summary := make(map[string]int)
	for _, category := range entity.GetValidCategories() {
		if count, exists := categoryCounts[category]; exists {
			summary[category] = count
		} else {
			summary[category] = 0
		}
	}

	return &CategorySummary{
		Categories: summary,
		Total:      total,
	}, nil
}

func (u *itemUsecase) PatchItem(ctx context.Context, id int64, input PatchItemInput) (*entity.Item, error) {
	if id <= 0 {
		return nil, domainErrors.ErrInvalidInput
	}

	item, err := u.itemRepo.FindByID(ctx, id)
	if err != nil {
		if domainErrors.IsNotFoundError(err) {
			return nil, domainErrors.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to retrieve item: %w", err)
	}

	if input.Name == nil && input.Brand == nil && input.PurchasePrice == nil {
		return nil, fmt.Errorf("%w: no fields to update", domainErrors.ErrInvalidInput)
	}

	//フィールド更新 nilをチェック
	if input.Name != nil {
		item.Name = *input.Name
	}

	if input.Brand != nil {
		item.Brand = *input.Brand
	}

	if input.PurchasePrice != nil {
		item.PurchasePrice = *input.PurchasePrice
	}

	item.UpdatedAt = time.Now()

	if err := item.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrInvalidInput, err.Error())
	}

	if err := u.itemRepo.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to update item: %w", err)
	}

	return item, nil
}
