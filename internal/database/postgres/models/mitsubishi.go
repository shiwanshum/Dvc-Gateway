package models

import (
	"time"
	"gorm.io/gorm"
)

type UUIDTimeStampedModelMixin struct {
	ID        string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type MitsubishiPlc struct {
	UUIDTimeStampedModelMixin

	FacilityName string `json:"facility_name"`
	Driver       string `json:"driver"`
	IpAddress    string `json:"ip_address"`
	ComType      string `json:"comtype"`
	Rack         int    `json:"rack"`
	Slot         int    `json:"slot"`
	Port         int    `json:"port"`
	WritePort    int    `json:"write_port"`
	AlarmPort    int    `json:"alarm_port"`
	Maker        string `json:"maker"`
}

type MitsubishiRobot struct {
	UUIDTimeStampedModelMixin
	PlcID     string              `json:"plc_id"`
	Plc       MitsubishiPlc       `gorm:"foreignKey:PlcID" json:"plc"`
	Name      string              `json:"name"`
	IpAddress string              `json:"ip_address"`
	ModelID   string              `json:"model_id"`
	Model     *MitsubishiRobotModel `gorm:"foreignKey:ModelID" json:"model"`
}

type MitsubishiRobotModel struct {
	UUIDTimeStampedModelMixin
	Manufacturer string `json:"manufacturer"`
	Name         string `json:"name"`
}

type MitsubishiTagList struct {
	UUIDTimeStampedModelMixin
	FacilityName		string          `json:"fac_name"`
	RobotID             string          `json:"robot_id"`
	Robot               MitsubishiRobot `gorm:"foreignKey:RobotID" json:"robot"`
	PlcID               string          `json:"plc_id"`
	Plc                 MitsubishiPlc   `gorm:"foreignKey:PlcID" json:"plc"`
	TagName             string          `json:"tag_name"` // e.g., D, M, X, Y
	TagAddress          string          `json:"tag_address"`
	Comment             string          `json:"comment"`
	DataType            string          `json:"data_type"`
	Action              string          `json:"action"`
	Screen              string          `json:"screen"`
	SvgElement          bool            `json:"svg_element"`
	TrueConditionColor  string          `json:"true_condition_color"`
	FalseConditionColor string          `json:"false_condition_color"`
	Blinking            bool            `json:"blinking"`
	RefreshRate         int             `json:"refresh_rate"`
}
