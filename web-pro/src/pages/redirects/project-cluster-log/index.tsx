import { Navigate } from '@umijs/max';

/** Legacy menu/component compatibility redirect */
export default function LegacyRedirect() {
  return <Navigate to="/project-services?tab=cluster-log" replace />;
}
