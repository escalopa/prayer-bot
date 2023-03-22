package handler

import objs "github.com/SakoDroid/telego/objects"

// admin is a wrapper for admin commands to check if the user is the bot owner
func (h *Handler) admin(command func(u *objs.Update)) func(u *objs.Update) {
	return func(u *objs.Update) {
		if u.Message.From.Id == h.botOwner {
			command(u)
		} else {
			h.Help(u)
		}
	}
}
