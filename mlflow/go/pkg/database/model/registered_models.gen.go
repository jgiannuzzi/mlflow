// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameRegisteredModel = "registered_models"

// RegisteredModel mapped from table <registered_models>
type RegisteredModel struct {
	Name            string `gorm:"column:name;primaryKey" json:"name"`
	CreationTime    int64  `gorm:"column:creation_time" json:"creation_time"`
	LastUpdatedTime int64  `gorm:"column:last_updated_time" json:"last_updated_time"`
	Description     string `gorm:"column:description" json:"description"`
}

// TableName RegisteredModel's table name
func (*RegisteredModel) TableName() string {
	return TableNameRegisteredModel
}
