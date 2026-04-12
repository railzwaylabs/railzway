package domain

type Country struct {
	Code string `json:"code" gorm:"type:char(2);primaryKey;column:code"`
	Name string `json:"name" gorm:"type:text;not null"`
}

func (Country) TableName() string { return "countries" }

type Timezone struct {
	Name string `json:"name" gorm:"type:text;primaryKey;column:name"`
}

func (Timezone) TableName() string { return "timezones" }

type Currency struct {
	Code string `json:"code" gorm:"type:char(3);primaryKey;column:code"`
	Name string `json:"name" gorm:"type:text;not null"`
}

func (Currency) TableName() string { return "currencies" }
