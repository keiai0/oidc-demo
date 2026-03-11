package totp

import (
	"testing"
	"time"
)

func Test_GenerateCode(t *testing.T) {
	// RFC 4226 Appendix D: テストベクタ（HOTP, SHA1, 6桁）
	// secret = "12345678901234567890" (ASCII)
	secret := []byte("12345678901234567890")

	tests := []struct {
		step     int64
		expected string
	}{
		{0, "755224"},
		{1, "287082"},
		{2, "359152"},
		{3, "969429"},
		{4, "338314"},
		{5, "254676"},
		{6, "287922"},
		{7, "162583"},
		{8, "399871"},
		{9, "520489"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := GenerateCode(secret, tt.step, 6)
			if got != tt.expected {
				t.Errorf("GenerateCode(step=%d) = %s, want %s", tt.step, got, tt.expected)
			}
		})
	}
}

func Test_GenerateCode_8digits(t *testing.T) {
	// RFC 6238 Appendix B: SHA1 テストベクタ（8桁）
	secret := []byte("12345678901234567890")

	tests := []struct {
		unixTime int64
		expected string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
		{20000000000, "65353130"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			step := tt.unixTime / 30
			got := GenerateCode(secret, step, 8)
			if got != tt.expected {
				t.Errorf("GenerateCode(time=%d, step=%d) = %s, want %s", tt.unixTime, step, got, tt.expected)
			}
		})
	}
}

func Test_Validate(t *testing.T) {
	secret := []byte("12345678901234567890")

	t.Run("valid code at current step", func(t *testing.T) {
		now := time.Unix(59, 0) // step=1
		code := GenerateCode(secret, 1, 6)

		step, valid := Validate(secret, code, now, 30, 6, nil)
		if !valid {
			t.Fatal("expected valid=true")
		}
		if step != 1 {
			t.Errorf("step = %d, want 1", step)
		}
	})

	t.Run("valid code at previous step (window)", func(t *testing.T) {
		now := time.Unix(60, 0) // step=2
		code := GenerateCode(secret, 1, 6) // step=1 のコード

		step, valid := Validate(secret, code, now, 30, 6, nil)
		if !valid {
			t.Fatal("expected valid=true (within window)")
		}
		if step != 1 {
			t.Errorf("step = %d, want 1", step)
		}
	})

	t.Run("valid code at next step (window)", func(t *testing.T) {
		now := time.Unix(30, 0) // step=1
		code := GenerateCode(secret, 2, 6) // step=2 のコード

		step, valid := Validate(secret, code, now, 30, 6, nil)
		if !valid {
			t.Fatal("expected valid=true (within window)")
		}
		if step != 2 {
			t.Errorf("step = %d, want 2", step)
		}
	})

	t.Run("invalid code", func(t *testing.T) {
		now := time.Unix(59, 0)

		_, valid := Validate(secret, "000000", now, 30, 6, nil)
		if valid {
			t.Fatal("expected valid=false for wrong code")
		}
	})

	t.Run("replay attack prevention", func(t *testing.T) {
		now := time.Unix(59, 0) // step=1
		code := GenerateCode(secret, 1, 6)

		lastStep := int64(1) // step=1 は既に使用済み
		_, valid := Validate(secret, code, now, 30, 6, &lastStep)
		if valid {
			t.Fatal("expected valid=false for replayed code")
		}
	})

	t.Run("replay allows newer step", func(t *testing.T) {
		now := time.Unix(90, 0) // step=3
		code := GenerateCode(secret, 3, 6)

		lastStep := int64(1) // step=1 が最後に使用された
		step, valid := Validate(secret, code, now, 30, 6, &lastStep)
		if !valid {
			t.Fatal("expected valid=true for newer step")
		}
		if step != 3 {
			t.Errorf("step = %d, want 3", step)
		}
	})

	t.Run("code outside window is rejected", func(t *testing.T) {
		now := time.Unix(120, 0) // step=4
		code := GenerateCode(secret, 1, 6) // step=1 のコード（window外）

		_, valid := Validate(secret, code, now, 30, 6, nil)
		if valid {
			t.Fatal("expected valid=false for code outside window")
		}
	})
}

func Test_GenerateSecret(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() error: %v", err)
	}
	if len(secret) != SecretSize {
		t.Errorf("secret length = %d, want %d", len(secret), SecretSize)
	}

	// 2つのシークレットが異なることを確認
	secret2, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() error: %v", err)
	}
	if string(secret) == string(secret2) {
		t.Error("two generated secrets should be different")
	}
}

func Test_BuildOTPAuthURI(t *testing.T) {
	secret := []byte("12345678901234567890")
	uri := BuildOTPAuthURI("OIDC Demo", "user@example.com", secret, 6, 30)

	if uri == "" {
		t.Fatal("URI should not be empty")
	}

	// URI が otpauth:// で始まることを確認
	if uri[:10] != "otpauth://" {
		t.Errorf("URI should start with otpauth://, got: %s", uri)
	}
}
