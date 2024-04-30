// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameExperiment = "experiments"

// Experiment mapped from table <experiments>
type Experiment struct {
	ExperimentID     int32           `gorm:"column:experiment_id;primaryKey;autoIncrement:true" json:"experiment_id"`
	Name             string          `gorm:"column:name;not null" json:"name"`
	ArtifactLocation string          `gorm:"column:artifact_location" json:"artifact_location"`
	LifecycleStage   string          `gorm:"column:lifecycle_stage" json:"lifecycle_stage"`
	CreationTime     int64           `gorm:"column:creation_time" json:"creation_time"`
	LastUpdateTime   int64           `gorm:"column:last_update_time" json:"last_update_time"`
	ExperimentTags   []ExperimentTag `gorm:"foreignKey:experiment_id;references:experiment_id" json:"experiment_tags"`
}

// TableName Experiment's table name
func (*Experiment) TableName() string {
	return TableNameExperiment
}
