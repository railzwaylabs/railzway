package domain

import "time"

type Country struct {
	Code      string    `json:"code" gorm:"type:char(2);primaryKey;column:code"`
	Name      string    `json:"name" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at,omitempty" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (Country) TableName() string { return "countries" }

type Timezone struct {
	Name      string    `json:"name" gorm:"type:text;primaryKey;column:name"`
	Region    string    `json:"region,omitempty" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at,omitempty" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (Timezone) TableName() string { return "timezones" }

type Currency struct {
	Code      string    `json:"code" gorm:"type:char(3);primaryKey;column:code"`
	Name      string    `json:"name" gorm:"type:text;not null"`
	Symbol    *string   `json:"symbol,omitempty" gorm:"type:text"`
	MinorUnit int16     `json:"minor_unit" gorm:"type:smallint;not null"`
	IsActive  bool      `json:"is_active,omitempty" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"created_at,omitempty" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (Currency) TableName() string { return "currencies" }

type CountryTimezone struct {
	CountryCode  string `json:"country_code" gorm:"type:char(2);primaryKey;column:country_code"`
	TimezoneName string `json:"timezone_name" gorm:"type:text;primaryKey;column:timezone_name"`
}

func (CountryTimezone) TableName() string { return "country_timezones" }
