package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/storage"
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/google/uuid"
)

type v interface {
}

var (
	ctx       context.Context
	ttsClient *texttospeech.Client
	gcsClient *storage.Client
)

// Deploy ke cloud tanpa harus simpan di file
func init() {
	// Instantiates a client.
	ctx := context.Background()

	var err error

	ttsClient, err = texttospeech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer ttsClient.Close()

	// Client Google Cloud Speech API inisialization
	gcsClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer gcsClient.Close()
}

func main() {
	// Instantiates a client.
	ctx := context.Background()

	textToSpeechClient, err := texttospeech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer textToSpeechClient.Close()

	// Client Google Cloud Speech API inisialization
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer gcsClient.Close()

	http.HandleFunc("/synthesized", synthesizedText(textToSpeechClient, gcsClient))
	http.HandleFunc("/", hc())
	http.ListenAndServe(":8080", nil)
}

// Cloud Function
func SynthesizedText(writer http.ResponseWriter, request *http.Request) {
	keys, ok := request.URL.Query()["text"]
	if !ok {
		log.Fatal(v("text not found"))
	}

	if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'key' is missing")
		return
	}

	// Query()["key"] will return an array of items,
	// we only want the single item.
	text := keys[0]

	ttsReq := textToSpeechRequest(text)

	resp, err := ttsClient.SynthesizeSpeech(request.Context(), ttsReq)
	if err != nil {
		log.Fatal(err)
	}

	// The resp's AudioContent is binary.
	id := uuid.New()
	filename := fmt.Sprintf("%s.mp3", id)
	err = os.WriteFile(filename, resp.AudioContent, 0644)
	if err != nil {
		log.Fatal(err)
	}
	// Make variable to hold the audio
	audiobuffer := bytes.NewReader(resp.AudioContent)

	gcsObj := gcsClient.Bucket("kiki-text-to-speech").Object(filename)

	// Create Golang API as an Uploader
	wc := gcsObj.NewWriter(request.Context())
	if _, err := io.Copy(wc, audiobuffer); err != nil {
		log.Fatalf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		log.Fatalf("Writer.Close: %v", err)
	}

	log.Println(writer, "Blob %v uploaded\n", filename)
	//End ============================= End

	// Make api public
	acl := gcsObj.ACL()
	if err := acl.Set(request.Context(), storage.AllUsers, storage.RoleReader); err != nil {
		log.Fatalf("ACLHandle.Set: %v", err)
	}
	log.Println(writer, "Blob %v is now publicly accessible.\n", filename)
	//End ============================= End

	//Get URL
	attrs, err := gcsObj.Attrs(request.Context())
	if err != nil {
		log.Fatalf("Object(%q).Attrs: %v", filename, err)
	}

	msg := fmt.Sprintf("Audio content written to file: %v\n", attrs.MediaLink)
	writer.Write([]byte(msg))

	respone := map[string]string{
		"url": attrs.MediaLink,
	}

	respBytes, err := json.Marshal(respone)
	if err != nil {
		log.Fatalf(err.Error())
	}
	writer.Header().Set(attrs.ContentType, "application/json")
	writer.Write(respBytes)
}

// HealthCheck
func hc() http.HandlerFunc {
	return func(writer http.ResponseWriter, respone *http.Request) {
		writer.Write([]byte("Hello"))
	}
}

// Handle Backend requests
func synthesizedText(ttsClient *texttospeech.Client, gcsClient *storage.Client) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		keys, ok := request.URL.Query()["text"]
		if !ok {
			log.Fatal(v("text not found"))
		}

		if !ok || len(keys[0]) < 1 {
			log.Println("Url Param 'key' is missing")
			return
		}

		// Query()["key"] will return an array of items,
		// we only want the single item.
		text := keys[0]

		ttsReq := textToSpeechRequest(text)

		resp, err := ttsClient.SynthesizeSpeech(request.Context(), ttsReq)
		if err != nil {
			log.Fatal(err)
		}

		// The resp's AudioContent is binary.
		id := uuid.New()
		filename := fmt.Sprintf("%s.mp3", id)
		// err = os.WriteFile(filename, resp.AudioContent, 0644)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// Make variable to hold the audio
		audiobuffer := bytes.NewReader(resp.AudioContent)

		gcsObj := gcsClient.Bucket("kiki-text-to-speech").Object(filename)

		// Create Golang API as an Uploader
		wc := gcsObj.NewWriter(request.Context())
		if _, err := io.Copy(wc, audiobuffer); err != nil {
			log.Fatalf("io.Copy: %v", err)
		}
		if err := wc.Close(); err != nil {
			log.Fatalf("Writer.Close: %v", err)
		}

		log.Println(writer, "Blob %v uploaded\n", filename)
		//End ============================= End

		// Make api public
		acl := gcsObj.ACL()
		if err := acl.Set(request.Context(), storage.AllUsers, storage.RoleReader); err != nil {
			log.Fatalf("ACLHandle.Set: %v", err)
		}
		log.Println(writer, "Blob %v is now publicly accessible.\n", filename)
		//End ============================= End

		//Get URL
		attrs, err := gcsObj.Attrs(request.Context())
		if err != nil {
			log.Fatalf("Object(%q).Attrs: %v", filename, err)
		}

		msg := fmt.Sprintf("Audio content written to file: %v\n", attrs.MediaLink)
		writer.Write([]byte(msg))

		respone := map[string]string{
			"url": attrs.MediaLink,
		}

		respBytes, err := json.Marshal(respone)
		if err != nil {
			log.Fatalf(err.Error())
		}
		writer.Header().Set(attrs.ContentType, "application/json")
		writer.Write(respBytes)
	}
}

// request from user
func textToSpeechRequest(text string) *texttospeechpb.SynthesizeSpeechRequest {
	// Perform the text-to-speech request on the text input with the selected
	// voice parameters and audio file type.
	return &texttospeechpb.SynthesizeSpeechRequest{
		// Set the text input to be synthesized.
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		// Build the voice request, select the language code ("en-US") and the SSML
		// voice gender ("neutral").
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "id-ID",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		},
		// Select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}
}
