package chapa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type BrevoEmailService struct {
	APIKey string
}

func NewBrevoEmailService() *BrevoEmailService {
	return &BrevoEmailService{
		APIKey: getRequiredEnv("API_KEY"),
	}
}

func (s *BrevoEmailService) SendOTP(
	email string,
	otp string,
) error {

	email = strings.TrimSpace(email)
	otp = strings.TrimSpace(otp)

	if email == "" || otp == "" {
		return fmt.Errorf("email or otp cannot be empty")
	}

	url := "https://api.brevo.com/v3/smtp/email"

	payload := map[string]interface{}{
		"sender": map[string]string{
			"name":  "JobConnect",
			"email": "nearbyme21@gmail.com",
		},

		"to": []map[string]string{
			{
				"email": email,
			},
		},

		"subject": "Your JobConnect OTP Code",

		"htmlContent": fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px;">
			<div style="max-width: 600px; margin: auto; background-color: #ffffff; padding: 30px; border-radius: 10px; box-shadow: 0 0 10px rgba(0,0,0,0.1);">

				<h2 style="color: #333333; text-align: center;">
					JobConnect Verification
				</h2>

				<p style="color: #555555; font-size: 16px; text-align: center;">
					Hi <strong>%s</strong>,<br>
					Use the OTP below to verify your account.
					This code is valid for <strong>15 minutes</strong>.
				</p>

				<div style="text-align: center; margin: 30px 0;">
					<span style="
						display: inline-block;
						padding: 15px 25px;
						font-size: 24px;
						font-weight: bold;
						color: #ffffff;
						background-color: #4CAF50;
						border-radius: 8px;
						letter-spacing: 2px;
					">
						%s
					</span>
				</div>

				<p style="color: #777777; font-size: 14px; text-align: center;">
					If you did not request this code,
					please ignore this email.
				</p>

				<hr style="border: none; border-top: 1px solid #eeeeee; margin: 20px 0;">

				<p style="color: #999999; font-size: 12px; text-align: center;">
					JobConnect • Your gateway to opportunities
				</p>

			</div>
		</div>
		`, email, otp),
	}

	jsonBody, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		url,
		bytes.NewBuffer(jsonBody),
	)

	if err != nil {
		return err
	}

	req.Header.Set("api-key", s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 &&
		resp.StatusCode != 201 {

		return fmt.Errorf(
			"failed to send otp: status code %d",
			resp.StatusCode,
		)
	}

	return nil
}

func getRequiredEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		panic(fmt.Sprintf(
			"missing env variable: %s",
			key,
		))
	}

	return value
}
