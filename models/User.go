package models

type User struct {
	Model
	Login         StringModel     `gorm:"type:varchar(255);default: null"`
	Password      StringModel     `gorm:"type:varchar(255);default: null"`
	Email         StringModel     `gorm:"type:varchar(255);default: null"`
	Phone         StringModel     `gorm:"type:varchar(255);default: null"`
	Name          StringModel     `gorm:"type:varchar(255);default: null"`
	Surname       StringModel     `gorm:"type:varchar(255);default: null"`
	Trans         StringModel     `gorm:"type:varchar(255);default: null"`
	Description   StringModel     `gorm:"type:text;default: null"`
	Type          IntModel        `gorm:"type:int;default: 0"`
	Status        IntModel        `gorm:"type:int;default: 0"`
	IdGoogle      StringModel     `gorm:"type:varchar(255);default: null"`
	IdGithub      StringModel     `gorm:"type:varchar(255);default: null"`
	IdBitbucket   StringModel     `gorm:"type:varchar(255);default: null"`
	IdVkontakte   StringModel     `gorm:"type:varchar(255);default: null"`
	IdFacebook    StringModel     `gorm:"type:varchar(255);default: null"`
	IdYandex      StringModel     `gorm:"type:varchar(255);default: null"`
	Gender        StringModel     `sql:"type:ENUM('male', 'female');default: null" gorm:"type:ENUM('male', 'female');default: null`
	CoordinateLat FloatModel      `gorm:"type:double;default: null"`
	CoordinateLng FloatModel      `gorm:"type:double;default: null"`
	Token         StringModel     `gorm:"type:varchar(255);default: null"`
	CreatedAt     TimeModel       `gorm:"type:timestamp;default: null"`
	UpdatedAt     TimeModel       `gorm:"type:timestamp;default: null"`
	DeletedAt     TimeDeleteModel `gorm:"index;type:timestamp;default: null"`
	OnlineAt      TimeModel       `gorm:"type:timestamp;default: null"`
	TokenAt       TimeModel       `gorm:"type:timestamp;default: null"`
}

func NewUser() *User {
	user := User{}

	return &user
}

func (User) TableName() string {
	return "users"
}

func (n User) isAdmin(u ...any) bool {
	return true
}
