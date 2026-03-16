import NextAuth, { NextAuthOptions } from "next-auth";
import CredentialsProvider from "next-auth/providers/credentials";

export const authOptions: NextAuthOptions = {
  providers: [
    CredentialsProvider({
      name: "Credentials",
      credentials: {
        email: { label: "Email", type: "text" },
        password: { label: "Password", type: "password" },
        captcha: { label: "Captcha", type: "text" },
        totp_code: { label: "2FA Code", type: "text" },
      },
      async authorize(credentials) {
        if (!credentials?.email || !credentials?.password) return null;

        try {
          const res = await fetch(process.env.NEXT_PUBLIC_API_URL ? `${process.env.NEXT_PUBLIC_API_URL}/api/auth/login` : "http://127.0.0.1:8080/api/auth/login", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              email: credentials.email,
              password: credentials.password,
              captcha: credentials.captcha,
              totp_code: credentials.totp_code,
            }),
          });

          const user = await res.json();
          
          if (res.ok && user && user.access_token) {
            // Expose the Golang backend access token globally to Next-Auth Session
            return {
              id: credentials.email, // placeholder ID
              email: credentials.email,
              accessToken: user.access_token,
            };
          }
          
          if (user.requires_2fa) {
            throw new Error("REQUIRES_2FA");
          } else if (res.status === 401 && user.captchaRequired) {
             throw new Error("CAPTCHA_REQUIRED");
          } else if (res.status === 403 && user.captchaRequired) {
             throw new Error("CAPTCHA_REQUIRED");
          } else if (res.status === 403) {
             throw new Error(user.error || "Email not verified");
          }

          throw new Error(user.error || "Login failed");
        } catch (e: any) {
          throw new Error(e.message);
        }
      },
    }),
  ],
  callbacks: {
    async jwt({ token, user }) {
      if (user) {
        token.accessToken = (user as any).accessToken;
      }
      return token;
    },
    async session({ session, token }) {
      (session as any).accessToken = token.accessToken;
      return session;
    },
  },
  pages: {
    signIn: "/login",
  },
  session: {
    strategy: "jwt",
  },
};

const handler = NextAuth(authOptions);
export { handler as GET, handler as POST };

