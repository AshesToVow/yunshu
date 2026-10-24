import { useRef } from "react";
import { newIdempotencyKey } from "../services/http";

/**
 * 高风险写操作的幂等键：组件挂载时生成一次，
 * 请求失败/超时重试复用同一个 key（服务端可安全重放），
 * 成功后调用 rotate() 换新 key，避免下一次真实提交被当作重放。
 */
export function useIdempotencyKey() {
  const ref = useRef<string>(newIdempotencyKey());
  const rotate = () => {
    ref.current = newIdempotencyKey();
  };
  return { key: ref.current, rotate };
}
