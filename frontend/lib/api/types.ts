export type ApiResponseCode =
  | "SUCCESS"
  | "BAD_REQUEST"
  | "UNAUTHORIZED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "INTERNAL_ERROR"
  | "VALIDATION_ERROR"
  | "TOKEN_EXPIRED"
  | "TOKEN_INVALID"
  | "RATE_LIMITED"
  | "SERVICE_UNAVAILABLE";

export type ApiEnvelope<T> = {
  code: ApiResponseCode | string;
  message: string;
  data?: T;
};

export type TokenResponse = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
};

export type UserResponse = {
  id: string;
  email: string;
  full_name?: string;
  role: string;
  email_verified: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type RegisterResponse = {
  user: UserResponse;
  requires_email_verification: boolean;
  message: string;
};

export type LoginResponse = {
  requires_2fa: boolean;
  temp_token?: string;
  message?: string;
  user?: UserResponse;
  token?: TokenResponse;
};

export type SessionResponse = {
  id: string;
  ip: string;
  user_agent: string;
  device_id: string;
  created_at: string;
  expires_at: string;
};

export type Setup2FAResponse = {
  secret: string;
  secret_url: string;
};

export type HealthResponse = {
  status: string;
  service: string;
};
