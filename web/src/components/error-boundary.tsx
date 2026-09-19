import { Button, Result, Typography } from "antd";
import { Component, type ErrorInfo, type ReactNode } from "react";
import i18n from "../i18n";

type Props = { children: ReactNode };

type State = { error: Error | null };

/**
 * 捕获子树渲染错误，避免整页白屏；便于运维平台在单页异常时仍可回到首页。
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // eslint-disable-next-line no-console
    console.error("ErrorBoundary", error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <Result
          status="error"
          title={i18n.t("app.renderErrorTitle")}
          subTitle={
            <Typography.Paragraph type="secondary" style={{ maxWidth: 560, margin: "0 auto" }}>
              {this.state.error.message || i18n.t("app.unknownError")}
            </Typography.Paragraph>
          }
          extra={
            <Button
              type="primary"
              onClick={() => {
                this.setState({ error: null });
                window.location.assign("/");
              }}
            >
              {i18n.t("app.backHome")}
            </Button>
          }
        />
      );
    }
    return this.props.children;
  }
}
