package model

import (
	"gorm.io/gorm"
)

type Student struct {
	Name string `column:"name"`
	Age  int    `column:"age"`
	Sex  string `column:"sex"`
	gorm.Model
}

type Teacher struct {
	Name    string `column:"name"`
	Project string `column:"project"`
	gorm.Model
}
