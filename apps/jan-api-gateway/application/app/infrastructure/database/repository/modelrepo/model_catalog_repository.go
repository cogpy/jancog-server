package modelrepo

import (
	"context"
	"errors"

	"gorm.io/gorm"
	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
)

type ModelCatalogGormRepository struct {
	db *transaction.Database
}

var _ domainmodel.ModelCatalogRepository = (*ModelCatalogGormRepository)(nil)

func NewModelCatalogGormRepository(db *transaction.Database) domainmodel.ModelCatalogRepository {
	return &ModelCatalogGormRepository{db: db}
}

func (repo *ModelCatalogGormRepository) applyFilter(query *gormgen.Query, sql gormgen.IModelCatalogDo, filter domainmodel.ModelCatalogFilter) gormgen.IModelCatalogDo {
	if filter.IDs != nil && len(*filter.IDs) > 0 {
		sql = sql.Where(query.ModelCatalog.ID.In((*filter.IDs)...))
	}
	if filter.PublicID != nil {
		sql = sql.Where(query.ModelCatalog.PublicID.Eq(*filter.PublicID))
	}
	if filter.IsModerated != nil {
		sql = sql.Where(query.ModelCatalog.IsModerated.Is(*filter.IsModerated))
	}
	if filter.Status != nil {
		sql = sql.Where(query.ModelCatalog.Status.Eq(string(*filter.Status)))
	}
	return sql
}

func (repo *ModelCatalogGormRepository) Create(ctx context.Context, catalog *domainmodel.ModelCatalog) error {
	model, err := dbschema.NewSchemaModelCatalog(catalog)
	if err != nil {
		return err
	}
	query := repo.db.GetQuery(ctx)
	if err := query.ModelCatalog.WithContext(ctx).Create(model); err != nil {
		return err
	}
	catalog.ID = model.ID
	catalog.CreatedAt = model.CreatedAt
	catalog.UpdatedAt = model.UpdatedAt
	catalog.Status = domainmodel.ModelCatalogStatus(model.Status)
	return nil
}

func (repo *ModelCatalogGormRepository) Update(ctx context.Context, catalog *domainmodel.ModelCatalog) error {
	model, err := dbschema.NewSchemaModelCatalog(catalog)
	if err != nil {
		return err
	}
	query := repo.db.GetQuery(ctx)
	return query.ModelCatalog.WithContext(ctx).Save(model)
}

func (repo *ModelCatalogGormRepository) DeleteByID(ctx context.Context, id uint) error {
	query := repo.db.GetQuery(ctx)
	_, err := query.ModelCatalog.WithContext(ctx).
		Where(query.ModelCatalog.ID.Eq(id)).
		Delete(&dbschema.ModelCatalog{})
	return err
}

func (repo *ModelCatalogGormRepository) FindByID(ctx context.Context, id uint) (*domainmodel.ModelCatalog, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.ModelCatalog.WithContext(ctx).Where(query.ModelCatalog.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD()
}

func (repo *ModelCatalogGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domainmodel.ModelCatalog, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.ModelCatalog.WithContext(ctx).Where(query.ModelCatalog.PublicID.Eq(publicID)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.EtoD()
}

func (repo *ModelCatalogGormRepository) FindByFilter(ctx context.Context, filter domainmodel.ModelCatalogFilter, p *query.Pagination) ([]*domainmodel.ModelCatalog, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ModelCatalog.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.Offset != nil && *p.Offset >= 0 {
			sql = sql.Offset(*p.Offset)
		}
		if p.Order == "desc" {
			sql = sql.Order(query.ModelCatalog.CreatedAt.Desc())
		} else {
			sql = sql.Order(query.ModelCatalog.CreatedAt.Asc())
		}
	}
	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := make([]*domainmodel.ModelCatalog, 0, len(rows))
	for _, item := range rows {
		domainItem, err := item.EtoD()
		if err != nil {
			return nil, err
		}
		result = append(result, domainItem)
	}
	return result, nil
}

func (repo *ModelCatalogGormRepository) Count(ctx context.Context, filter domainmodel.ModelCatalogFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.ModelCatalog.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	return sql.Count()
}
