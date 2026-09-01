import { Navigate } from '@umijs/max';

/** Legacy menu/component compatibility redirect */
export default function LegacyRedirect() {
  return <Navigate to="/workflow/definitions?domain=dbmgmt" replace />;
}
