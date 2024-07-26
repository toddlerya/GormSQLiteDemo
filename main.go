package main

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/toddlerya/sqliteRWSplit/loader"
	"github.com/toddlerya/sqliteRWSplit/model"
)

func main() {

	fmt.Println("Hello World")
	model.InitDB(true)
	size := 10 * 1000
	studentDatas := loader.GenerateStudentList(size)
	logrus.Infof("生成数据: %d", len(studentDatas))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		logrus.Info("In Load Task1")
		defer wg.Done()
		loader.InsertStudentList(studentDatas)
	}()

	var task1ErrorCount int = 0
	wg.Add(1)
	go func() {
		for i := 1; i <= size; i++ {
			var student model.Student
			// err := model.ReadDB.Exec("select * from student where id = ?", i).Error
			err := model.ReadDB.Take(&student, i).Error
			if err != nil {
				logrus.Errorf("IN Query Task1 查询错误: id=%d, ERROR: %v", i, err)
				task1ErrorCount = task1ErrorCount + 1
			} else {
				logrus.Info("IN Query Task1", student)
			}
		}
		defer wg.Done()
	}()

	var task2ErrorCount int = 0
	wg.Add(1)
	go func() {
		for i := 1; i <= size; i++ {
			var student model.Student
			err := model.ReadDB.Take(&student, i).Error
			if err != nil {
				logrus.Errorf("IN Query Task2 查询错误: id=%d, ERROR: %v", i, err)
				task2ErrorCount = task2ErrorCount + 1
			} else {
				logrus.Info("IN Query Task2", student)
			}
		}
		defer wg.Done()
	}()

	wg.Wait()
	logrus.Infof("task1ErrorCount: %d task2ErrorCount: %d ", task1ErrorCount, task2ErrorCount)
}
