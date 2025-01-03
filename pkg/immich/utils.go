package immich

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func parseResponse[T any](resp *http.Response, object *T) error {
	if resp != nil {
		if resp.StatusCode >= 400 {
			if resp.Body != nil {
				defer resp.Body.Close()

				sr := ServerResponse{}
				if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
					return fmt.Errorf("can't decode server error: %w", err)
				}

				return fmt.Errorf("%s: %s", resp.Status, sr.Message)
			}
		}

		if resp.Body != nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusNoContent {
				return nil
			}
			return json.NewDecoder(resp.Body).Decode(object)
		}
	}

	return nil
}
