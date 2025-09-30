package service

import "awesomeProject/internal/repository"

func GetUserByID(id string) (interface{}, error) {
	return repository.FindUserByID(id)
}
