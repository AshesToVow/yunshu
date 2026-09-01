// @ts-nocheck
import { Divider, type DividerProps } from "antd";

/** 带左侧标题的分节线（web antd 5 使用 orientation；sync 至 web-pro 时自动转为 titlePlacement） */
export function SectionDivider({ children, ...props }: DividerProps) {
  return (
    <Divider titlePlacement="start" {...props}>
      {children}
    </Divider>
  );
}
