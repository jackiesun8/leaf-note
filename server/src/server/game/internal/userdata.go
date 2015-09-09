package internal

import (
	"fmt"
)

type UserData struct {
	UserID int "_id"
	AccID  string
}

func (data *UserData) initValue(accID string) error {
	userID, err := mongoDBNextSeq("users")
	if err != nil {
		return fmt.Errorf("get next users id error: %v", err)
	}

	data.UserID = userID
	data.AccID = accID

	return nil
}
