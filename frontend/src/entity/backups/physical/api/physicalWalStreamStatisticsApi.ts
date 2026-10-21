import { getApplicationServer } from '../../../../constants';
import { apiHelper } from '../../../../shared/api/apiHelper';
import type { WalStreamLiveResponse } from '../model/WalStreamLiveResponse';
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

  // Read every second while the tab is visible, so a failed request is simply replaced by the next
  // one instead of being retried.
  async getWalStreamLive(databaseId: string) {
    return apiHelper.fetchGetJson<WalStreamLiveResponse>(
      `${getApplicationServer()}/api/v1/backups/physical/database/${databaseId}/wal-stream/live`,
    );
  },
};
