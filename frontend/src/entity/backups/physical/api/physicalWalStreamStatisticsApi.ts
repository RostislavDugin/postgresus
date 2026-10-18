import { getApplicationServer } from '../../../../constants';
import { apiHelper } from '../../../../shared/api/apiHelper';
import type { WalStreamStatistics } from '../model/WalStreamStatistics';
import type { WalStreamStatisticsPeriod } from '../model/WalStreamStatisticsPeriod';

export const physicalWalStreamStatisticsApi = {
  // No retry inside the request: the Monitoring tab polls every 30 seconds, and a request that kept
  // retrying for a minute and a half would overlap the next poll and hide a failure the tab must
  // show.
  async getWalStreamStatistics(databaseId: string, period: WalStreamStatisticsPeriod) {
    const params = new URLSearchParams({ period });

    return apiHelper.fetchGetJson<WalStreamStatistics>(
      `${getApplicationServer()}/api/v1/backups/physical/database/${databaseId}/wal-stream/statistics?${params.toString()}`,
    );
  },
};
