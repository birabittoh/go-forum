package cache

import (
	"fmt"
	"goforum/internal/models"
	"strings"
)

const (
	UserIDKey   = "id:%d"
	NameKey     = "name:%s"
	EmailKey    = "email:%s"
	UsernameKey = "username:%s"
)

func updateUserCache(c *Cache, user *models.User) {
	c.users.Add(fmt.Sprintf(UserIDKey, user.ID), user)
	c.users.Add(fmt.Sprintf(NameKey, strings.ToLower(user.Email)), user)
	c.users.Add(fmt.Sprintf(NameKey, strings.ToLower(user.Username)), user)
	c.users.Add(fmt.Sprintf(EmailKey, strings.ToLower(user.Email)), user)
	c.users.Add(fmt.Sprintf(UsernameKey, strings.ToLower(user.Username)), user)
}

func (c *Cache) GetUserByID(userID uint) (user models.User, ok bool) {
	key := fmt.Sprintf(UserIDKey, userID)
	userP, ok := c.users.Get(key)
	if ok {
		return *userP, true
	}

	err := c.db.First(&user, userID).Error
	if err != nil {
		return
	}

	updateUserCache(c, &user)
	return user, true
}

func (c *Cache) GetUserByName(username string) (user models.User, ok bool) {
	name := strings.ToLower(username)
	key := fmt.Sprintf(NameKey, name)
	userP, ok := c.users.Get(key)
	if ok {
		return *userP, true
	}

	err := c.db.Where("LOWER(username) = ? OR LOWER(email) = ?", name).First(&user).Error
	if err != nil {
		return
	}

	updateUserCache(c, &user)
	return user, true
}

func (c *Cache) GetUserByUsername(username string) (user models.User, ok bool) {
	name := strings.ToLower(username)
	key := fmt.Sprintf(UsernameKey, name)
	userP, ok := c.users.Get(key)
	if ok {
		return *userP, true
	}

	err := c.db.Where("LOWER(username) = ?", name).First(&user).Error
	if err != nil {
		return
	}

	updateUserCache(c, &user)
	return user, true
}

func (c *Cache) GetUserByEmail(email string) (user models.User, ok bool) {
	mail := strings.ToLower(email)
	key := fmt.Sprintf(EmailKey, mail)
	userP, ok := c.users.Get(key)
	if ok {
		return *userP, true
	}

	err := c.db.Where("LOWER(email) = ?", mail).First(&user).Error
	if err != nil {
		return
	}

	updateUserCache(c, &user)
	return user, true
}

func (c *Cache) CreateUser(user *models.User) error {
	err := c.db.Create(user).Error
	if err != nil {
		return err
	}

	countAllUsers, ok := c.counts.Get(CountsKeyAllUsers)
	if ok {
		c.counts.Add(CountsKeyAllUsers, countAllUsers+1)
	}

	updateUserCache(c, user)
	return nil
}

func (c *Cache) UpdateUser(user *models.User) error {
	err := c.db.Save(user).Error
	if err != nil {
		return err
	}

	updateUserCache(c, user)
	return nil
}

func (c *Cache) DeleteUser(user *models.User) error {
	err := c.db.Delete(user).Error
	if err != nil {
		return err
	}

	countAllUsers, ok := c.counts.Get(CountsKeyAllUsers)
	if ok {
		c.counts.Add(CountsKeyAllUsers, countAllUsers-1)
	}

	c.users.Remove(fmt.Sprintf(UserIDKey, user.ID))
	c.users.Remove(fmt.Sprintf(NameKey, strings.ToLower(user.Email)))
	c.users.Remove(fmt.Sprintf(NameKey, strings.ToLower(user.Username)))
	c.users.Remove(fmt.Sprintf(EmailKey, strings.ToLower(user.Email)))
	c.users.Remove(fmt.Sprintf(UsernameKey, strings.ToLower(user.Username)))

	return nil
}
