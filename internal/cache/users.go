package cache

import (
	"goforum/internal/models"
	"strings"
)

func updateUserCache(c *Cache, user *models.User) {
	c.usernameToID[strings.ToLower(user.Username)] = user.ID
	c.emailToID[strings.ToLower(user.Email)] = user.ID
	c.users.Add(user.ID, user)
}

func (c *Cache) GetUserByID(userID uint) (user models.User, ok bool) {
	userP, ok := c.users.Get(userID)
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
	user, ok = c.GetUserByUsername(username)
	if ok {
		return
	}

	return c.GetUserByEmail(username)
}

func (c *Cache) GetUserByUsername(username string) (user models.User, ok bool) {
	name := strings.ToLower(username)

	id, exists := c.usernameToID[name]
	if exists {
		return c.GetUserByID(id)
	}

	err := c.db.Where("LOWER(username) = ?", name).First(&user).Error
	if err != nil {
		return
	}

	c.usernameToID[name] = user.ID
	updateUserCache(c, &user)
	return user, true
}

func (c *Cache) GetUserByEmail(email string) (user models.User, ok bool) {
	mail := strings.ToLower(email)

	id, exists := c.emailToID[mail]
	if exists {
		return c.GetUserByID(id)
	}

	err := c.db.Where("LOWER(email) = ?", mail).First(&user).Error
	if err != nil {
		return
	}

	c.emailToID[mail] = user.ID
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

	c.users.Remove(user.ID)
	delete(c.usernameToID, strings.ToLower(user.Username))
	delete(c.emailToID, strings.ToLower(user.Email))

	return nil
}
