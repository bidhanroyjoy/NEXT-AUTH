import axios from 'axios';
import { getSession } from 'next-auth/react';

const api = axios.create({
  baseURL: 'http://localhost:8080/api', // Backend base URL
  withCredentials: true, // Crucial for sending HttpOnly Refresh token & reading CSRF
});

// Add a request interceptor
api.interceptors.request.use(
  async (config) => {
    // 1. Ensure GET request to Golang triggers CSRF generation if not present
    // Let's assume frontend calls health API once on load to establish CSRF

    // 2. Attach Next.js Session Access Token (JWT)
    // Needs to work in browser. 
    if (typeof window !== 'undefined') {
        const session: any = await getSession();
        if (session?.accessToken) {
            config.headers.Authorization = `Bearer ${session.accessToken}`;
        }

        // 3. Attach CSRF matching token
        const match = document.cookie.match(new RegExp('(^| )csrf_token=([^;]+)'));
        if (match) {
            config.headers['X-CSRF-Token'] = match[2];
        }
    }

    return config;
  },
  (error) => Promise.reject(error)
);

// Add a response interceptor for 401 Refresh Token
api.interceptors.response.use(
  (response) => {
    return response;
  },
  async (error) => {
    const originalRequest = error.config;
    
    // Ignore refresh route to prevent infinite loop
    if (originalRequest.url === '/auth/refresh') {
        return Promise.reject(error);
    }

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      try {
        // Request token refresh (Backend checks HttpOnly cookie automatically appended)
        const res = await axios.post('http://localhost:8080/api/auth/refresh', {}, { withCredentials: true });

        if (res.status === 200 && res.data.access_token) {
          // Retry original request with newly obtained access_token
          originalRequest.headers.Authorization = `Bearer ${res.data.access_token}`;
          return api(originalRequest);
        }
      } catch (refreshError) {
        // Token refresh failed completely. Redirect to login
        if (typeof window !== 'undefined') {
            window.location.href = '/login';
        }
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);

export default api;
