# Authentication Application (Next.js + Golang + PostgreSQL)

This is a comprehensive full-stack authentication system demonstrating modern security practices.

## Technologies Used
*   **Frontend**: Next.js (App Router), TailwindCSS, Next-Auth v4, Axios.
*   **Backend**: Golang (Gin Framework), GORM, PostgreSQL.
*   **Security**: JWT (Access + Refresh Tokens wrapped in HttpOnly cookies), CSRF Protection (Double Submit Cookie Pattern), Bcrypt.

## Features Implemented
1.  **Registration & Email OTP**: Users sign up with an email and password. An OTP is "sent" via mock email logging, which must be verified before login is permitted.
2.  **Login with Captcha**: After 3 failed login attempts, an additional `captcha` field is required to successfully authenticate (Mocked Captcha hint provided in UI).
3.  **JWT Tokens & Refreshing**: Access Tokens (15m expiry) are stored statefully by NextAuth, while Refresh Tokens (7d expiry) are securely stored as HTTPOnly cookies. An Axios interceptor automatically refreshes the Access Token upon 401 Unauthorized responses.
4.  **CSRF Protection**: The Golang API relies on a strict CORS policy combined with CSRF validation for all state-mutating requests (POST, PUT, DELETE).
5.  **Forgot/Reset Password**: Complete flow for password recovery using an OTP sent to the user's email.

## Setup Instructions

### 1. Database (PostgreSQL)
Ensure you have a PostgreSQL database running. Create a `.env` file in the `backend/` directory:
```env
DB_HOST=localhost
DB_USER=your_postgres_user
DB_PASSWORD=your_postgres_password
DB_NAME=auth_db
DB_PORT=5432
JWT_ACCESS_SECRET=super_secret_access_key
JWT_REFRESH_SECRET=super_secret_refresh_key
```

### 2. Backend (Golang)
Navigate to the `backend` folder:
```bash
go mod tidy
go run *.go
# Or build and run
go build -o auth-backend
./auth-backend
```
The server will start on `http://localhost:8080`. The logs will display "mock emails" for OTPs.

### 3. Frontend (Next.js)
Navigate to the `frontend` folder. Create a `.env.local`:
```env
NEXTAUTH_URL=http://localhost:3000
NEXTAUTH_SECRET=a_very_secure_random_string_here
```
Run the application:
```bash
npm install
npm run dev
```
The application will be accessible at `http://localhost:3000/login`.

## Code Walkthrough

*   `backend/utils.go`: Houses standalone logic for bcrypt hashing, securely generating 6-digit OTPs using `crypto/rand`, and creating JWTs using the HS256 algorithm.
*   `backend/handlers.go` & `handlers_password.go`: API controllers. Every endpoint meticulously validates structural input data (binding JSON) before querying the Postgres instance using GORM.
*   `backend/middleware.go`: Custom implementations for evaluating the HTTPOnly CSRF Cookie against an `X-CSRF-Token` header mapping, alongside the access token `RequireAuth` enforcer.
*   `frontend/src/app/api/auth/[...nextauth]/route.ts`: Contains the `CredentialsProvider` integration logic specifically tailored to interface intelligently with the Golang API's 403 HTTP error shapes, triggering Captcha UI updates correctly.
*   `frontend/src/lib/axios.ts`: Handles the complex synchronization between client-side data (NextAuth session) and cross-origin security rules (`withCredentials`, CSRF extractions).
