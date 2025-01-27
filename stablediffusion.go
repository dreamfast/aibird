package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yunginnanet/girc-atomic"
)

// Txt2Img API Request
type (
	StableDiffusionRequest struct {
		EnableHr          bool                   `json:"enable_hr"`
		DenoisingStrength int                    `json:"denoising_strength"`
		FirstphaseWidth   int                    `json:"firstphase_width"`
		FirstphaseHeight  int                    `json:"firstphase_height"`
		Prompt            string                 `json:"prompt"`
		Styles            []string               `json:"styles"`
		Seed              int                    `json:"seed"`
		Subseed           int                    `json:"subseed"`
		SubseedStrength   int                    `json:"subseed_strength"`
		SeedResizeFromH   int                    `json:"seed_resize_from_h"`
		SeedResizeFromW   int                    `json:"seed_resize_from_w"`
		BatchSize         int                    `json:"batch_size"`
		NIter             int                    `json:"n_iter"`
		Steps             int                    `json:"steps"`
		CfgScale          float32                `json:"cfg_scale"`
		Width             int                    `json:"width"`
		Height            int                    `json:"height"`
		RestoreFaces      bool                   `json:"restore_faces"`
		RefinerCheckpoint string                 `json:"refiner_checkpoint"`
		RefinerSwitchAt   float32                `json:"refiner_switch_at"`
		Tiling            bool                   `json:"tiling"`
		NegativePrompt    string                 `json:"negative_prompt"`
		Eta               int                    `json:"eta"`
		SChurn            int                    `json:"s_churn"`
		STmax             int                    `json:"s_tmax"`
		STmin             int                    `json:"s_tmin"`
		SNoise            int                    `json:"s_noise"`
		OverrideSettings  map[string]interface{} `json:"override_settings"`
		SamplerIndex      string                 `json:"sampler_index"`
	}

	// Txt2Img API response
	StableDiffusionResponse struct {
		Images []string `json:"images"`
	}
)

func sdRequest(c *girc.Client, e girc.Event, prompt string) {
	posturl := config.StableDiffusion.Host + "/sdapi/v1/txt2img"

	sendToIrc(c, e, "Processing Stable Diffusion: "+prompt+"...")

	// Bad words for bad chatters
	if safetyFilter(prompt) {
		prompt = config.StableDiffusion.BadWordsPrompt
	}

	// new variable that converts prompt to slug
	slug := cleanFileName(prompt)

	// Create a new struct
	sd := StableDiffusionRequest{
		EnableHr: false,
		// DenoisingStrength: 0,
		// FirstphaseWidth:   0,
		// FirstphaseHeight:  0,
		Prompt: prompt,
		Seed:   -1,
		// Subseed:           -1,
		// SubseedStrength:   0,
		// SeedResizeFromH:   -1,
		// SeedResizeFromW:   -1,
		BatchSize:    1,
		NIter:        1,
		Steps:        config.StableDiffusion.Steps,
		CfgScale:     config.StableDiffusion.CfgScale,
		Width:        config.StableDiffusion.Width,
		Height:       config.StableDiffusion.Height,
		RestoreFaces: config.StableDiffusion.RestoreFace,
		// Tiling:            false,
		// NegativePrompt:    config.StableDiffusion.NegativePrompt,
		// RefinerCheckpoint: "sd_xl_refiner_1.0.safetensors",
		// RefinerSwitchAt: 0.65,
		// Eta:               0,
		// SChurn:            0,
		// STmax:             0,
		// STmin:             0,
		// SNoise:            1,
		SamplerIndex: config.StableDiffusion.Sampler,
	}

	// Prepare sd for http NewRequest
	reqStr, err := json.Marshal(sd)
	if err != nil {
		sendToIrc(c, e, err.Error())
		log.Println(err.Error())
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", posturl, strings.NewReader(string(reqStr)))

	if err != nil {
		log.Println(err.Error())
		sendToIrc(c, e, err.Error())
		return
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Println(err.Error())
		sendToIrc(c, e, "There as an error processing your request, the SD host may be down or had issues with vram and your request.")
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err.Error())
		sendToIrc(c, e, "There as an error processing your request, the SD host may be down or had issues with vram and your request.")
		return
	}

	post := &StableDiffusionResponse{}
	err = json.Unmarshal(body, post)

	if err != nil {
		log.Println(err.Error())
		sendToIrc(c, e, "There as an error processing your request, the SD host may be down or had issues with vram and your request.")
		return
	}

	if res.StatusCode != http.StatusOK {
		sendToIrc(c, e, fmt.Sprint(res.StatusCode))
		return
	}

	// generate random string
	randValue := rand.Int63n(10000)

	// generate a random value with length of 4
	randValue = randValue % 10000
	fileName := slug + "_" + strconv.FormatInt(randValue, 4) + ".png"

	// decode base64 image and save to fileName
	decoded, err := base64.StdEncoding.DecodeString(post.Images[0])
	if err != nil {
		log.Println(err.Error())
		sendToIrc(c, e, err.Error())
		return
	}

	err = ioutil.WriteFile(fileName, decoded, 0644)
	if err != nil {
		log.Println(err.Error())
		sendToIrc(c, e, err.Error())
		return
	}

	// append the current pwd to fileName
	fileName = filepath.Base(fileName)

	// download image
	content := fileHole("https://filehole.org/", fileName)
	sendToIrc(c, e, e.Source.Name+": "+content)
}

func sdAdmin(c *girc.Client, e girc.Event, message string) {
	// remove sd from message and trim
	message = strings.TrimSpace(strings.TrimPrefix(message, "sd"))
	parts := strings.SplitN(message, " ", 2)

	switch parts[0] {
	case "vars":
		sendToIrc(c, e, "Stable Diffusion Vars: ")
		sendToIrc(c, e, fmt.Sprintf("%+v", config.StableDiffusion))
		return

	case "set":
		message = strings.TrimSpace(strings.TrimPrefix(message, "set"))
		parts := strings.SplitN(message, " ", 2)

		// update config.StableDiffusion key parts[0] with value parts[1]
		switch parts[0] {
		case "steps":
			// convert parts[1] to int
			steps, err := strconv.Atoi(parts[1])
			if err != nil {
				sendToIrc(c, e, err.Error())
				return
			}

			// update config
			config.StableDiffusion.Steps = steps
			sendToIrc(c, e, "Updated sd steps to: "+strconv.Itoa(config.StableDiffusion.Steps))

		case "width":
			// convert parts[1] to int
			width, err := strconv.Atoi(parts[1])
			if err != nil {
				sendToIrc(c, e, err.Error())
				return
			}

			// update config
			config.StableDiffusion.Width = width
			sendToIrc(c, e, "Updated sd width to: "+strconv.Itoa(config.StableDiffusion.Width))

		case "height":
			// convert parts[1] to int
			height, err := strconv.Atoi(parts[1])
			if err != nil {
				sendToIrc(c, e, err.Error())
				return
			}

			// update config
			config.StableDiffusion.Height = height
			sendToIrc(c, e, "Updated sd height to: "+strconv.Itoa(config.StableDiffusion.Height))

		case "sampler":
			if parts[1] != "DDIM" && parts[1] != "Euler a" && parts[1] != "Euler" {
				sendToIrc(c, e, "Invalid sampler, must be 'DDIM', 'Euler a' or 'Euler'")
				return
			}

			config.StableDiffusion.Sampler = parts[1]
			sendToIrc(c, e, "Updated sd sampler to: "+config.StableDiffusion.Sampler)

		case "NegativePrompt":
			// update config.StableDiffusion.NegativePrompt
			config.StableDiffusion.NegativePrompt = parts[1]
			sendToIrc(c, e, "Updated sd negativePrompt to: "+config.StableDiffusion.NegativePrompt)

		case "cfg":
			// convert string parts[1] to float32
			cfg, err := strconv.ParseFloat(parts[1], 32)
			if err != nil {
				sendToIrc(c, e, err.Error())
				return
			}

			config.StableDiffusion.CfgScale = float32(cfg)
			sendToIrc(c, e, "Updated sd cfg to: "+parts[1])
		}
	}
}

// This is why we can't have nice things
func safetyFilter(message string) bool {
	for _, word := range config.StableDiffusion.BadWords {
		if strings.Contains(strings.ToLower(message), strings.ToLower(word)) {
			return true
		}
	}

	return false
}
