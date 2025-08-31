package command

import (
	"context"
	"net/http"

	"meowabot/internal/tools/media"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type MessageOptions struct {
	QuotedMessage *events.Message
	Caption       *string
	FileName      *string
	MentionedJid  []string

	Seconds         *uint32
	Mimetype        *string
	ExternalAdReply *waProto.ContextInfo_ExternalAdReplyInfo
}

func (ctx *CommandContext) Reply(text string) {
	message := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: &text,
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      &ctx.Msg.Info.ID,
				Participant:   proto.String(ctx.Msg.Info.Sender.String()),
				QuotedMessage: ctx.Msg.Message,
			},
		},
	}
	_, err := ctx.Client.SendMessage(context.TODO(), ctx.Msg.Info.Chat, message)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("Error sending text message")
	}
}

func (ctx *CommandContext) DeleteMessage(chatID types.JID, senderJID types.JID, message *events.Message) {
	_, err := ctx.Client.SendMessage(context.TODO(), chatID, ctx.Client.BuildRevoke(chatID, senderJID, message.Info.ID))
	if err != nil {
		ctx.Log.Error().Err(err).Str("ChatID", chatID.String()).Str("MessageID", message.Info.ID).Msg("Failed to delete message")
	}
}

func (ctx *CommandContext) SendTextMessage(to types.JID, text string, msgExtras *MessageOptions) {
	message := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        &text,
			ContextInfo: &waProto.ContextInfo{},
		},
	}
	if msgExtras != nil {
		message.ExtendedTextMessage.ContextInfo.ExternalAdReply = msgExtras.ExternalAdReply
		message.ExtendedTextMessage.ContextInfo.MentionedJID = msgExtras.MentionedJid

		if msgExtras.QuotedMessage != nil {
			message.ExtendedTextMessage.ContextInfo.StanzaID = &msgExtras.QuotedMessage.Info.ID
			message.ExtendedTextMessage.ContextInfo.Participant = proto.String(msgExtras.QuotedMessage.Info.Sender.String())
			message.ExtendedTextMessage.ContextInfo.QuotedMessage = msgExtras.QuotedMessage.Message
		}
	}

	_, err := ctx.Client.SendMessage(context.TODO(), to, message)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("Error sending text message")
	}
}

func (ctx *CommandContext) ReactMessage(message *events.Message, emoji string) {

	var err error

	if message.Info.Chat.Server == types.NewsletterServer {
		err = ctx.Client.NewsletterSendReaction(message.Info.Chat, message.Info.ServerID, emoji, ctx.Client.GenerateMessageID())
	} else {
		_, err = ctx.Client.SendMessage(context.TODO(), message.Info.Chat, ctx.Client.BuildReaction(message.Info.Chat, message.Info.Sender, message.Info.ID, emoji))
	}
	if err != nil {
		ctx.Log.Error().Err(err).Str("to", message.Info.Chat.String()).Msg("Error sending react message")
	}
}

func (ctx *CommandContext) SendImageMessage(to types.JID, data []byte, msgExtras *MessageOptions) {
	uploaded, err := ctx.Client.Upload(context.TODO(), data, whatsmeow.MediaImage)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("Error uploading image")
		return
	}

	thumbnail, err := media.ResizeImg(data, 74, 74)
	if err != nil {
		ctx.Log.Warn().Err(err).Msg("Failed to generate image thumbnail")
	}

	message := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(http.DetectContentType(data)),
			FileEncSHA256: uploaded.FileEncSHA256,
			JPEGThumbnail: thumbnail,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			ContextInfo:   &waProto.ContextInfo{},
		},
	}
	if msgExtras != nil {
		message.ImageMessage.Caption = msgExtras.Caption
		message.ImageMessage.ContextInfo.ExternalAdReply = msgExtras.ExternalAdReply
		message.ImageMessage.ContextInfo.MentionedJID = msgExtras.MentionedJid

		if msgExtras.QuotedMessage != nil {
			message.ImageMessage.ContextInfo.StanzaID = &msgExtras.QuotedMessage.Info.ID
			message.ImageMessage.ContextInfo.Participant = proto.String(msgExtras.QuotedMessage.Info.Sender.String())
			message.ImageMessage.ContextInfo.QuotedMessage = msgExtras.QuotedMessage.Message
		}
		if msgExtras.Mimetype != nil {
			message.ImageMessage.Mimetype = msgExtras.Mimetype
		}
	}

	_, err = ctx.Client.SendMessage(context.TODO(), to, message)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("Error sending image message")
	}

}

func (ctx *CommandContext) SendVideoMessage(to types.JID, data []byte, msgExtras *MessageOptions) {
	uploaded, err := ctx.Client.Upload(context.TODO(), data, whatsmeow.MediaVideo)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("Error uploading video")
		return
	}

	var thumbnail []byte
	thumbnail, err = media.GetVideoThumbnail(data)
	if err != nil {
		ctx.Log.Warn().Err(err).Msg("Failed to generate video thumbnail")
	}

	message := &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL:           proto.String(uploaded.URL),
			Mimetype:      proto.String(http.DetectContentType(data)),
			JPEGThumbnail: thumbnail,
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			ContextInfo:   &waProto.ContextInfo{},
		},
	}

	if msgExtras != nil {
		message.VideoMessage.ContextInfo.ExternalAdReply = msgExtras.ExternalAdReply
		message.VideoMessage.ContextInfo.MentionedJID = msgExtras.MentionedJid
		message.VideoMessage.Caption = msgExtras.Caption

		if msgExtras.QuotedMessage != nil {
			message.VideoMessage.ContextInfo.StanzaID = &msgExtras.QuotedMessage.Info.ID
			message.VideoMessage.ContextInfo.Participant = proto.String(msgExtras.QuotedMessage.Info.Sender.String())
			message.VideoMessage.ContextInfo.QuotedMessage = msgExtras.QuotedMessage.Message
		}
		if msgExtras.Mimetype != nil {
			message.VideoMessage.Mimetype = msgExtras.Mimetype
		}
	}

	_, err = ctx.Client.SendMessage(context.TODO(), to, message)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("error sending video message")
	}

}

func (ctx *CommandContext) SendDocumentMessage(to types.JID, data []byte, msgExtras *MessageOptions) {
	uploaded, err := ctx.Client.Upload(context.TODO(), data, whatsmeow.MediaDocument)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("Error uploading document")
		return
	}

	message := &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			FileName:      proto.String("file"),
			Mimetype:      proto.String(http.DetectContentType(data)),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			ContextInfo:   &waProto.ContextInfo{},
		},
	}

	if msgExtras != nil {
		message.DocumentMessage.ContextInfo.ExternalAdReply = msgExtras.ExternalAdReply
		message.DocumentMessage.ContextInfo.MentionedJID = msgExtras.MentionedJid

		message.DocumentMessage.ContextInfo.MentionedJID = msgExtras.MentionedJid
		message.DocumentMessage.Caption = msgExtras.Caption

		if msgExtras.QuotedMessage != nil {
			message.DocumentMessage.ContextInfo.StanzaID = &msgExtras.QuotedMessage.Info.ID
			message.DocumentMessage.ContextInfo.Participant = proto.String(msgExtras.QuotedMessage.Info.Sender.String())
			message.DocumentMessage.ContextInfo.QuotedMessage = msgExtras.QuotedMessage.Message
		}
		if msgExtras.Mimetype != nil {
			message.DocumentMessage.Mimetype = msgExtras.Mimetype
		}
		if msgExtras.FileName != nil {
			message.DocumentMessage.FileName = msgExtras.FileName
		}
		if msgExtras.Caption != nil {
			message.DocumentMessage.Caption = msgExtras.Caption
		}
	}

	_, err = ctx.Client.SendMessage(context.TODO(), to, message)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("error sending video message")
	}

}

func (ctx *CommandContext) SendStickerMessage(to types.JID, data []byte, msgExtras *MessageOptions) {
	uploaded, err := ctx.Client.Upload(context.TODO(), data, whatsmeow.MediaImage)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("Error uploading sticker")
		return
	}

	message := &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String("image/webp"),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			ContextInfo:   &waProto.ContextInfo{},
		},
	}
	if msgExtras != nil {
		message.StickerMessage.ContextInfo.ExternalAdReply = msgExtras.ExternalAdReply
		message.StickerMessage.ContextInfo.MentionedJID = msgExtras.MentionedJid

		if msgExtras.QuotedMessage != nil {
			message.StickerMessage.ContextInfo.StanzaID = &msgExtras.QuotedMessage.Info.ID
			message.StickerMessage.ContextInfo.Participant = proto.String(msgExtras.QuotedMessage.Info.Sender.String())
			message.StickerMessage.ContextInfo.QuotedMessage = msgExtras.QuotedMessage.Message
		}
	}

	_, err = ctx.Client.SendMessage(context.TODO(), to, message)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("Error sending sticker message")
	}
}

func (ctx *CommandContext) SendAudioMessage(to types.JID, data []byte, msgExtras *MessageOptions) {
	uploaded, err := ctx.Client.Upload(context.TODO(), data, whatsmeow.MediaAudio)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("Error uploading audio")
		return
	}

	message := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String("audio/mp4"),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			ContextInfo:   &waProto.ContextInfo{},
		},
	}

	if msgExtras != nil {
		message.AudioMessage.ContextInfo.ExternalAdReply = msgExtras.ExternalAdReply
		message.AudioMessage.ContextInfo.MentionedJID = msgExtras.MentionedJid
		message.AudioMessage.Seconds = msgExtras.Seconds

		if msgExtras.QuotedMessage != nil {
			message.AudioMessage.ContextInfo.StanzaID = &msgExtras.QuotedMessage.Info.ID
			message.AudioMessage.ContextInfo.Participant = proto.String(msgExtras.QuotedMessage.Info.Sender.String())
			message.AudioMessage.ContextInfo.QuotedMessage = msgExtras.QuotedMessage.Message
		}
		if msgExtras.Mimetype != nil {
			message.AudioMessage.Mimetype = msgExtras.Mimetype
		}
	}

	if message.AudioMessage.Seconds == nil {
		d, _ := media.GetAudioDuration(data)
		message.AudioMessage.Seconds = proto.Uint32(d)
	}

	_, err = ctx.Client.SendMessage(context.TODO(), to, message)
	if err != nil {
		ctx.Log.Error().Err(err).Msg("Error sending audio message")
	}
}
