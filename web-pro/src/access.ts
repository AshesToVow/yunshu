/**
 * @see https://umijs.org/docs/max/access#access
 * */
export default function access(initialState: { currentUser?: API.CurrentUser } | undefined) {
  const { currentUser } = initialState ?? {};
  const isAdmin =
    currentUser?.access === 'admin' ||
    (currentUser as YunshuAPI.UserItem | undefined)?.roles?.some((r: YunshuAPI.RoleItem) => r.code === 'super-admin');
  return {
    canAdmin: Boolean(isAdmin),
  };
}
