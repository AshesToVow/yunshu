/**
 * Yunshu 后端 API 代理（仅 dev）
 */
export default {
  dev: {
    '/api/': {
      target: 'http://127.0.0.1:8080',
      changeOrigin: true,
      ws: true,
    },
    '/swagger/': {
      target: 'http://127.0.0.1:8080',
      changeOrigin: true,
    },
  },
  test: {
    '/api/': {
      target: 'http://127.0.0.1:8080',
      changeOrigin: true,
    },
  },
  pre: {
    '/api/': {
      target: 'http://127.0.0.1:8080',
      changeOrigin: true,
    },
  },
};
