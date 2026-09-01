import dayjs from 'dayjs';

const numberFormatter = new Intl.NumberFormat('en-US');

/** Format a number with thousand separators. */
export const formatNumber = (val: number | string): string => {
  const parsed = Number(val);
  return Number.isFinite(parsed) ? numberFormatter.format(parsed) : '';
};

/** Format a number as yuan currency string. */
export const formatYuan = (val: number | string) => `¥ ${formatNumber(val)}`;

export function formatDateTime(value?: string | null) {
  if (!value) {
    return '-';
  }
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss');
}

export function statusText(status: number) {
  return status === 1 ? '启用' : '停用';
}

export function statusColor(status: number) {
  return status === 1 ? 'success' : 'default';
}
