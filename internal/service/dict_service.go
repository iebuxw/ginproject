package service

import (
	"errors"
	"ginproject/internal/dao"
	"ginproject/internal/model"
	"time"
)

type DictTypeService struct {
	dictTypeDAO *dao.DictTypeDAO
	dictDataDAO *dao.DictDataDAO
}

func NewDictTypeService(d *dao.DictTypeDAO, dd *dao.DictDataDAO) *DictTypeService {
	return &DictTypeService{dictTypeDAO: d, dictDataDAO: dd}
}

func (s *DictTypeService) Create(dt *model.DictType) error {
	now := model.DateTime(time.Now())
	dt.CreatedAt = now
	dt.UpdatedAt = now
	return s.dictTypeDAO.Create(dt)
}

func (s *DictTypeService) Update(dt *model.DictType) error {
	dt.UpdatedAt = model.DateTime(time.Now())
	return s.dictTypeDAO.Update(dt)
}

func (s *DictTypeService) Delete(id uint) error {
	has, err := s.dictDataDAO.HasData(id)
	if err != nil {
		return err
	}
	if has {
		return errors.New("该字典类型下存在数据项，无法删除")
	}
	return s.dictTypeDAO.Delete(id)
}

func (s *DictTypeService) FindByID(id uint) (*model.DictType, error) { return s.dictTypeDAO.FindByID(id) }

func (s *DictTypeService) FindByCode(code string) (*model.DictType, error) {
	return s.dictTypeDAO.FindByCode(code)
}

func (s *DictTypeService) FindPage(page, pageSize int, keyword string) ([]model.DictType, int64, error) {
	return s.dictTypeDAO.FindPage(page, pageSize, keyword)
}

// ---- DictData Service ----

type DictDataService struct {
	dictDataDAO *dao.DictDataDAO
}

func NewDictDataService(d *dao.DictDataDAO) *DictDataService {
	return &DictDataService{dictDataDAO: d}
}

func (s *DictDataService) Create(dd *model.DictData) error {
	now := model.DateTime(time.Now())
	dd.CreatedAt = now
	dd.UpdatedAt = now
	return s.dictDataDAO.Create(dd)
}

func (s *DictDataService) Update(dd *model.DictData) error {
	dd.UpdatedAt = model.DateTime(time.Now())
	return s.dictDataDAO.Update(dd)
}

func (s *DictDataService) Delete(id uint) error { return s.dictDataDAO.Delete(id) }

func (s *DictDataService) FindByID(id uint) (*model.DictData, error) {
	return s.dictDataDAO.FindByID(id)
}

func (s *DictDataService) FindByDictTypeID(dictTypeID uint) ([]model.DictData, error) {
	return s.dictDataDAO.FindByDictTypeID(dictTypeID)
}

func (s *DictDataService) FindPage(page, pageSize int, dictTypeID uint) ([]model.DictData, int64, error) {
	return s.dictDataDAO.FindPage(page, pageSize, dictTypeID)
}

func (s *DictDataService) FindByDictTypeCode(code string) ([]model.DictData, error) {
	return s.dictDataDAO.FindByDictTypeCode(code)
}
