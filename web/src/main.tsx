import React from "react";
import ReactDOM from "react-dom/client";
import "./i18n";
import { App } from "./app/app";
import "./styles/global.css";
import "./styles/yunshu-design-system.css";
import "./styles/ops-tokens.css";
import "./styles/yunshu-impeccable-polish.css";
import "./styles/yunshu-ops-global.css";
import "./styles/yunshu-auth.css";
import "antd/dist/reset.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
