package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
	"gmauleon.org/purr/pkg/discord"
	"gmauleon.org/purr/pkg/immich"
	"go.uber.org/zap"
)

const (
	environmentVariablePrefix = "PURR"
	interactionName           = "Backup"
	internalErrorStatus       = "internal error"
)

var (
	discordAppID             string
	discordToken             string
	discordAuthorizedUserIDs []string

	immichURL    string
	immichAPIKey string

	cachePath string
	logger    *zap.Logger
)

var rootCmd = &cobra.Command{
	Use:   "purr",
	Short: "Purr is a discord bot",
	RunE: func(cmd *cobra.Command, args []string) error {
		return launch()
	},
}

func Execute() {
	statusCode := 0
	if err := rootCmd.Execute(); err != nil {
		statusCode = 1
	}

	_ = logger.Sync()
	os.Exit(statusCode)
}

func init() {
	logger = zap.Must(zap.NewProduction())

	rootCmd.PersistentFlags().StringVar(&discordAppID, "discord-app-id", "", "Discord application ID")
	rootCmd.PersistentFlags().StringVar(&discordToken, "discord-token", "", "Discord token")
	rootCmd.PersistentFlags().StringSliceVar(&discordAuthorizedUserIDs, "discord-authorized-user-ids", []string{}, "Discord authorized users IDs")
	rootCmd.PersistentFlags().StringVar(&immichURL, "immich-url", "", "Immich URL")
	rootCmd.PersistentFlags().StringVar(&immichAPIKey, "immich-api-key", "", "Immich API key")
	rootCmd.PersistentFlags().StringVar(&cachePath, "cache-path", "", "Cache path to temporarily store images")
}

func launch() error {
	if err := verifyFlags(); err != nil {
		return err
	}

	// Create Discord bot
	bot, err := discord.NewBot(logger, discordAppID, discordToken)
	if err != nil {
		return fmt.Errorf("failed to create discord bot: %w", err)
	}

	// Create Immich client
	immichClient, err := immich.NewClient(immichURL, immichAPIKey)
	if err != nil {
		return fmt.Errorf("failed to create immich client: %w", err)
	}

	if err := bot.AddInteraction(interactionName, discordAuthorizedUserIDs, createImmichCallback(context.Background(), immichClient, cachePath)); err != nil {
		return fmt.Errorf("failed to add interaction :%w", err)
	}

	if err := bot.Start(); err != nil {
		return fmt.Errorf("failed to start bot: %w", err)
	}

	// Block the program from exiting
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	if err := bot.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown bot: %w", err)
	}

	return nil
}

func verifyFlags() error {
	var flagErrors error

	discordAppID = os.Getenv(environmentVariablePrefix + "_DISCORD_APP_ID")
	if discordAppID == "" {
		flagErrors = multierror.Append(flagErrors, errors.New("discord-app-id is required"))
	}

	discordToken = os.Getenv(environmentVariablePrefix + "_DISCORD_TOKEN")
	if discordToken == "" {
		flagErrors = multierror.Append(flagErrors, errors.New("discord-token is required"))
	}

	discordAuthorizedUserIDs = strings.Split(os.Getenv(environmentVariablePrefix+"_DISCORD_AUTHORIZED_USER_IDS"), ",")
	if discordAuthorizedUserIDs[0] == "" {
		flagErrors = multierror.Append(flagErrors, errors.New("discord-authorized-user-ids is required"))
	}

	immichURL = os.Getenv(environmentVariablePrefix + "_IMMICH_URL")
	if immichURL == "" {
		flagErrors = multierror.Append(flagErrors, errors.New("immich-url is required"))
	}

	immichAPIKey = os.Getenv(environmentVariablePrefix + "_IMMICH_API_KEY")
	if immichAPIKey == "" {
		flagErrors = multierror.Append(flagErrors, errors.New("immich-api-key is required"))
	}

	cachePath = os.Getenv(environmentVariablePrefix + "_CACHE_PATH")
	if cachePath == "" {
		flagErrors = multierror.Append(flagErrors, errors.New("cache-path is required"))
	}

	return flagErrors
}

// Function to download the image
func createImmichCallback(ctx context.Context, immichClient *immich.Client, cachePath string) discord.InteractionCallback {
	return func(filename string, url string, messageTime time.Time) (string, error) {
		fullPath := filepath.Join(cachePath, filename)

		// Create a file to save the image
		out, err := os.Create(fullPath)
		if err != nil {
			return fmt.Sprintf("%s: %s\n", filename, internalErrorStatus), fmt.Errorf("failed to create file: %w", err)
		}
		defer out.Close()
		defer os.Remove(fullPath)

		// Get the image from the URL
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Sprintf("%s: %s\n", filename, internalErrorStatus), fmt.Errorf("failed http get: %w", err)
		}
		defer resp.Body.Close()

		// Write the image to the file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return fmt.Sprintf("%s: %s\n", filename, internalErrorStatus), fmt.Errorf("failed to write image: %w", err)
		}

		immichResponse, err := immichClient.UploadAsset(ctx, fullPath, messageTime, messageTime)
		if err != nil {
			return fmt.Sprintf("%s: %s\n", filename, internalErrorStatus), fmt.Errorf("failed to upload asset: %w", err)
		}

		return fmt.Sprintf("%s: %s\n", filename, immichResponse.Status), nil
	}
}
