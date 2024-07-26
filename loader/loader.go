package loader

import (
	"github.com/Pallinder/go-randomdata"
	"github.com/sirupsen/logrus"
	"github.com/toddlerya/sqliteRWSplit/model"
)

func GenerateStudent() *model.Student {
	return &model.Student{
		Name: randomdata.FullName(randomdata.RandomGender),
		Age:  randomdata.Number(1, 100),
		Sex:  "男",
	}
}

func GenerateStudentList(n int) []*model.Student {
	var students []*model.Student
	for i := 0; i < n; i++ {
		students = append(students, GenerateStudent())
	}
	return students
}

func InsertStudent(student *model.Student) {
	model.WriteDB.Create(student)
}

func InsertStudentList(students []*model.Student) {

	if err := model.WriteDB.CreateInBatches(students, 1000).Error; err != nil {
		logrus.Error(err)
	}
}

func GenerateTeacher() *model.Teacher {
	return &model.Teacher{
		Name:    randomdata.FullName(randomdata.RandomGender),
		Project: randomdata.City(),
	}
}

func GenerateTeacherList(n int) []*model.Teacher {
	var teachers []*model.Teacher
	for i := 0; i < n; i++ {
		teachers = append(teachers, GenerateTeacher())
	}
	return teachers
}

func InsertTeacher(teacher *model.Teacher) {
	model.WriteDB.Create(teacher)
}

func InsertTeacherList(teachers []*model.Teacher) {
	model.WriteDB.CreateInBatches(teachers, 100)
}
