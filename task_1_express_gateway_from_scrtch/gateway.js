const express = require("express");
const morgan = require("morgan");
const jwt = require("jsonwebtoken");
const rateLimit = require("express-rate-limit");
const { createProxyMiddleware } = require("http-proxy-middleware");

const app = express();

const SECRET = "mysecret";

// ---------------- Logging ----------------

app.use(morgan("dev"));

// ---------------- Rate Limiting ----------------

app.use(
    rateLimit({
        windowMs: 60 * 1000,
        max: 100,
    })
);

// ---------------- JWT Authentication ----------------

function verifyJWT(req, res, next) {

    // Skip login endpoint
    if (req.path.startsWith("/auth/login")) {
        return next();
    }

    const auth = req.headers.authorization;

    if (!auth)
        return res.status(401).json({
            message: "Token Missing"
        });

    const token = auth.split(" ")[1];

    try {

        const user = jwt.verify(token, SECRET);

        req.user = user;

        next();

    } catch {

        return res.status(401).json({
            message: "Invalid Token"
        });

    }
}

app.use(verifyJWT);

// ---------------- Proxy : Auth Service ----------------

app.use(
    "/auth",
    createProxyMiddleware({
        target: "http://localhost:8001",
        changeOrigin: true,
    })
);

// ---------------- Proxy : Maps Service ----------------

app.use(
    "/maps",
    createProxyMiddleware({
        target: "http://localhost:8002",
        changeOrigin: true,
        onProxyReq(proxyReq, req) {

            // Forward user info

            if (req.user) {

                proxyReq.setHeader(
                    "x-user-id",
                    req.user.id
                );

                proxyReq.setHeader(
                    "x-role",
                    req.user.role
                );

            }

        }
    })
);

// ---------------- Proxy : Payment Service ----------------

app.use(
    "/payment",
    createProxyMiddleware({
        target: "http://localhost:8003",
        changeOrigin: true,
    })
);

// ---------------- Error ----------------

app.use((err, req, res, next) => {

    console.error(err);

    res.status(500).json({
        message: "Gateway Error"
    });

});

app.listen(8080, () => {
    console.log("Gateway running on 8080");
});