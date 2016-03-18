package margelet

import (
	"gopkg.in/telegram-bot-api.v2"
)

func handleUpdate(margelet *Margelet, update tgbotapi.Update) {
	defer func() {
		if err := recover(); err != nil {
			margelet.QuickSend(update.Message.Chat.ID, "Panic occured!")
		}
	}()

	if message := update.Message; message.Text != "" {
		margelet.ChatRepository.Add(message.Chat.ID)

		// If we have active session in this chat with this user, handle it first
		if command := margelet.SessionRepository.Command(message.Chat.ID, message.From.ID); len(command) > 0 {
			// TODO: /cancel command should cancel any active session!
			margelet.HandleSession(message, command)
		} else {
			if message.IsCommand() {
				handleCommand(margelet, message)
			} else {
				handleMessage(margelet, message)
			}
		}
	} else {
		handleInline(margelet, update.InlineQuery)
	}
}

func handleInline(margelet *Margelet, query tgbotapi.InlineQuery) {
	handler := margelet.InlineHandler

	if handler != nil {
		handler.HandleInline(margelet, query)
	}
}

func handleCommand(margelet *Margelet, message tgbotapi.Message) {
	if authHandler, ok := margelet.CommandHandlers[message.Command()]; ok {
		if err := authHandler.Allow(message); err != nil {
			margelet.QuickSend(message.Chat.ID, "Authorization error: "+err.Error())
			return
		}
		err := authHandler.handler.HandleCommand(margelet, message)

		if err != nil {
			margelet.QuickSend(message.Chat.ID, "Error occured: "+err.Error())
		}
		return
	}

	if authHandler, ok := margelet.SessionHandlers[message.Command()]; ok {
		margelet.SessionRepository.Create(message.Chat.ID, message.From.ID, message.Command())
		handleSession(margelet, message, authHandler)
		return
	}
}

func handleMessage(margelet *Margelet, message tgbotapi.Message) {
	for _, handler := range margelet.MessageHandlers {
		err := handler.HandleMessage(margelet, message)

		if err != nil {
			margelet.QuickSend(message.Chat.ID, "Error occured: "+err.Error())
		}
	}
}

func handleSession(margelet *Margelet, message tgbotapi.Message, authHandler authorizedSessionHandler) {
	if err := authHandler.Allow(message); err != nil {
		margelet.QuickSend(message.Chat.ID, "Authorization error: "+err.Error())
		return
	}
	finish, err := authHandler.handler.HandleSession(margelet, message, margelet.SessionRepository.Dialog(message.Chat.ID, message.From.ID))
	if finish {
		margelet.SessionRepository.Remove(message.Chat.ID, message.From.ID)
		return
	}

	if err == nil {
		margelet.SessionRepository.Add(message.Chat.ID, message.From.ID, message)
	}
}