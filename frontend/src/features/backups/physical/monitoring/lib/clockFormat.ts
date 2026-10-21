import { getUserTimeFormat } from '../../../../../shared/time';

/* eslint-disable i18next/no-literal-string -- dayjs format tokens, not copy */
export const getClockFormat = (): string => (getUserTimeFormat().use12Hours ? 'h:mm A' : 'HH:mm');
/* eslint-enable i18next/no-literal-string */
