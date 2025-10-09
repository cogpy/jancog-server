package modelrepo

import (
	"context"

	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
)

type ProviderModelGormRepository struct {
	db *transaction.Database
}

var _ domainmodel.ProviderModelRepository = (*ProviderModelGormRepository)(nil)

func NewProviderModelGormRepository(db *transaction.Database) domainmodel.ProviderModelRepository {
	return &ProviderModelGormRepository{db: db}
}

func (repo *ProviderModelGormRepository) applyFilter(query *gormgen.Query, sql gormgen.IProviderModelDo, filter domainmodel.ProviderModelFilter) gormgen.IProviderModelDo {
	if filter.IDs != nil && len(*filter.IDs) > 0 {
		sql = sql.Where(query.ProviderModel.ID.In((*filter.IDs)...))
	}
	if filter.ProviderID != nil {
		sql = sql.Where(query.ProviderModel.ProviderID.Eq(*filter.ProviderID))
	}
	if filter.ProviderIDs != nil && len(*filter.ProviderIDs) > 0 {
		sql = sql.Where(query.ProviderModel.ProviderID.In((*filter.ProviderIDs)...))
	}
	if filter.ModelCatalogID != nil {
		sql = sql.Where(query.ProviderModel.ModelCatalogID.Eq(*filter.ModelCatalogID))
	}
	if filter.ModelKey != nil {
		sql = sql.Where(query.ProviderModel.ModelKey.Eq(*filter.ModelKey))
	}
	if filter.ModelKeys != nil && len(*filter.ModelKeys) > 0 {
		sql = sql.Where(query.ProviderModel.ModelKey.In((*filter.ModelKeys)...))
	}
	if filter.Active != nil {
		sql = sql.Where(query.ProviderModel.Active.Is(*filter.Active))
	}
	if filter.SupportsImages != nil {
		sql = sql.Where(query.ProviderModel.SupportsImages.Is(*filter.SupportsImages))
	}
	if filter.SupportsEmbeddings != nil {
		sql = sql.Where(query.ProviderModel.SupportsEmbeddings.Is(*filter.SupportsEmbeddings))
	}
	if filter.SupportsReasoning != nil {
		sql = sql.Where(query.ProviderModel.SupportsReasoning.Is(*filter.SupportsReasoning))
	}
	return sql
}

func (repo *ProviderModelGormRepository) Create(ctx context.Context, model *domainmodel.ProviderModel) error {
	schemaModel, err := dbschema.NewSchemaProviderModel(model)
	if err != nil {
		return err
	}
	query := repo.db.GetQuery(ctx)
	if err := query.ProviderModel.WithContext(ctx).Create(schemaModel); err != nil {
		return err
	}
	model.ID = schemaModel.ID
	model.CreatedAt = schemaModel.CreatedAt
	model.UpdatedAt = schemaModel.UpdatedAt
	return nil
}

func (repo *ProviderModelGormRepository) Update(ctx context.Context, model *domainmodel.ProviderModel) error {
	schemaModel, err := dbschema.NewSchemaProviderModel(model)
	if err != nil {
		return err
	}
	query := repo.db.GetQuery(ctx)
	return query.ProviderModel.WithContext(ctx).Save(schemaModel)
}

func (repo *ProviderModelGormRepository) DeleteByID(ctx context.Context, id uint) error {
	query := repo.db.GetQuery(ctx)
	_, err := query.ProviderModel.WithContext(ctx).Where(query.ProviderModel.ID.Eq(id)).Delete(&dbschema.ProviderModel{})
	return err
}

func (repo *ProviderModelGormRepository) FindByID(ctx context.Context, id uint) (*domainmodel.ProviderModel, error) {
	query := repo.db.GetQuery(ctx)
	schemaModel, err := query.ProviderModel.WithContext(ctx).Where(query.ProviderModel.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return schemaModel.EtoD()
}

func (repo *ProviderModelGormRepository) FindByFilter(ctx context.Context, filter domainmodel.ProviderModelFilter, p *query.Pagination) ([]*domainmodel.ProviderModel, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ProviderModel.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.Offset != nil && *p.Offset >= 0 {
			sql = sql.Offset(*p.Offset)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(query.ProviderModel.ID.Lt(*p.After))
			} else {
				sql = sql.Where(query.ProviderModel.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(query.ProviderModel.ID.Desc())
		} else {
			sql = sql.Order(query.ProviderModel.ID.Asc())
		}
	}
	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := make([]*domainmodel.ProviderModel, 0, len(rows))
	for _, item := range rows {
		domainItem, err := item.EtoD()
		if err != nil {
			return nil, err
		}
		result = append(result, domainItem)
	}
	return result, nil
}

func (repo *ProviderModelGormRepository) Count(ctx context.Context, filter domainmodel.ProviderModelFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ProviderModel.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	return sql.Count()
}
