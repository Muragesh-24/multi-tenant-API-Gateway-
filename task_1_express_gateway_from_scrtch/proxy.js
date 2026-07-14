const express = require("express");
const { createProxyMiddleware } = require("http-proxy-middleware");

const app = express();

app.use(
  "/auth",
  createProxyMiddleware({
    target: "http://localhost:8001",
    changeOrigin: true,
  })
);

app.use(
  "/maps",
  createProxyMiddleware({
    target: "http://localhost:8002",
    changeOrigin: true,
  })
);

app.listen(8080);