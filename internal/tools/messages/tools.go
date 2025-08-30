package messages

import (
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func GetMentionedJIDS(message *waE2E.Message) []string {
	if message == nil {
		return nil
	}

	if m := message.DocumentWithCaptionMessage.GetMessage(); m != nil {
		return GetMentionedJIDS(m)
	}
	if m := message.GroupStatusMentionMessage.GetMessage(); m != nil {
		return GetMentionedJIDS(m)
	}
	if m := message.GroupStatusMessage.GetMessage(); m != nil {
		return GetMentionedJIDS(m)
	}
	if m := message.GroupMentionedMessage.GetMessage(); m != nil {
		return GetMentionedJIDS(m)
	}
	if m := message.RequestPaymentMessage.GetNoteMessage(); m != nil {
		return GetMentionedJIDS(m)
	}
	if m := message.ViewOnceMessage.GetMessage(); m != nil {
		return GetMentionedJIDS(m)
	}
	if m := message.ViewOnceMessageV2.GetMessage(); m != nil {
		return GetMentionedJIDS(m)
	}

	contexts := []func() *waE2E.ContextInfo{
		message.AudioMessage.GetContextInfo,
		message.ButtonsResponseMessage.GetContextInfo,
		message.ContactMessage.GetContextInfo,
		message.ContactsArrayMessage.GetContextInfo,
		message.DocumentMessage.GetContextInfo,
		message.ExtendedTextMessage.GetContextInfo,
		message.ImageMessage.GetContextInfo,
		message.LiveLocationMessage.GetContextInfo,
		message.LocationMessage.GetContextInfo,
		message.OrderMessage.GetContextInfo,
		message.ProductMessage.GetContextInfo,
		message.PtvMessage.GetContextInfo,
		message.RequestPhoneNumberMessage.GetContextInfo,
		message.VideoMessage.GetContextInfo,
	}

	for _, getContext := range contexts {
		if m := getContext(); m != nil {
			if m.MentionedJID != nil {
				return m.MentionedJID
			}
			return []string{}
		}
	}

	return []string{}
}

func GetMessageText(message *waE2E.Message) (text string, isValid bool) {
	if message == nil {
		return "", false
	}

	recursives := []func() *waE2E.Message{
		message.GroupMentionedMessage.GetMessage,
		message.GroupStatusMentionMessage.GetMessage,
		message.GroupStatusMessage.GetMessage,
		message.RequestPaymentMessage.GetNoteMessage,
		message.ViewOnceMessage.GetMessage,
		message.ViewOnceMessageV2.GetMessage,
		message.ViewOnceMessageV2Extension.GetMessage,
	}

	for _, getMsg := range recursives {
		if m := getMsg(); m != nil {
			text, _ := GetMessageText(m)
			if message.ViewOnceMessage != nil || message.ViewOnceMessageV2 != nil {
				return text, true
			}
			return text, false
		}
	}

	switch {

	case message.Conversation != nil:
		return message.GetConversation(), true
	case message.ExtendedTextMessage != nil:
		return message.ExtendedTextMessage.GetText(), true
	case message.ImageMessage != nil:
		return message.ImageMessage.GetCaption(), true
	case message.VideoMessage != nil:
		return message.VideoMessage.GetCaption(), true
	case message.PtvMessage != nil:
		return message.PtvMessage.GetCaption(), true
	case message.DocumentMessage != nil:
		return message.DocumentMessage.GetCaption(), true

	case message.LiveLocationMessage != nil:
		return message.LiveLocationMessage.GetCaption(), false
	case message.OrderMessage != nil:
		return message.OrderMessage.GetMessage(), false
	case message.ProductMessage != nil:
		return message.ProductMessage.GetBody(), false
	}

	return "", false
}

func GetQuotedJid(m *events.Message) (jid types.JID, err error) {
	if m.Message.ExtendedTextMessage != nil {
		if m.Message.ExtendedTextMessage.ContextInfo.Participant != nil {
			jidString := *m.Message.ExtendedTextMessage.ContextInfo.Participant
			jid, err = types.ParseJID(jidString)
			if err != nil {
				return
			}
		} else if len(m.Message.ExtendedTextMessage.ContextInfo.MentionedJID) > 0 {
			jidString := m.Message.ExtendedTextMessage.ContextInfo.MentionedJID[0]
			jid, err = types.ParseJID(jidString)
			if err != nil {
				return
			}
		}
	}
	return
}

func GetImageMessage(m *events.Message) *waE2E.Message {
	if m.Message.ImageMessage != nil {
		return m.Message
	}
	if m.Message.ExtendedTextMessage != nil && m.Message.ExtendedTextMessage.GetContextInfo().GetQuotedMessage().GetImageMessage() != nil {
		return m.Message.ExtendedTextMessage.GetContextInfo().GetQuotedMessage()
	}
	return nil
}

func GetVideoMessage(m *events.Message) *waE2E.Message {
	if m.Message.VideoMessage != nil {
		return m.Message
	}
	if m.Message.ExtendedTextMessage != nil && m.Message.ExtendedTextMessage.GetContextInfo().GetQuotedMessage().GetVideoMessage() != nil {
		return m.Message.ExtendedTextMessage.GetContextInfo().GetQuotedMessage()
	}
	return nil
}

func GetStickerMessage(m *events.Message) *waE2E.Message {
	if m.Message.StickerMessage != nil {
		return m.Message
	}
	if m.Message.ExtendedTextMessage != nil && m.Message.ExtendedTextMessage.GetContextInfo().GetQuotedMessage().GetStickerMessage() != nil {
		return m.Message.ExtendedTextMessage.GetContextInfo().GetQuotedMessage()
	}
	return nil
}
