package physical_service

import (
	"time"

	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
)

// GetWalStreamStatistics reads the samples, the streamer row and the newest
// archived segment of one database and decides its verdict, keeping-up summary
// and chart series at query.Now. The verdict reads the last hour of samples,
// the series reads the requested period.
func (s *PhysicalBackupService) GetWalStreamStatistics(query WalStreamStatisticsQuery) (*WalStreamStatistics, error) {
	newestSample, err := s.walStreamSampleRepository.FindNewestByDatabase(query.DatabaseID)
	if err != nil {
		return nil, err
	}

	hourAgo := query.Now.Add(-time.Hour)

	recentSamples, err := s.walStreamSampleRepository.FindSinceByDatabase(query.DatabaseID, hourAgo)
	if err != nil {
		return nil, err
	}

	sessionStartsLastHour, err := s.walStreamSampleRepository.CountSessionStartsSince(query.DatabaseID, hourAgo)
	if err != nil {
		return nil, err
	}

	streamer, err := s.walStreamerRepository.FindByDatabaseID(query.DatabaseID)
	if err != nil {
		return nil, err
	}

	newestCommittedSegment, err := s.walSegmentRepository.FindLatestCommittedSegment(query.DatabaseID)
	if err != nil {
		return nil, err
	}

	seriesFrom := physical_repositories.GetFirstSeriesBucketAt(query.Now, query.Period, query.BucketDuration)
	bucketCount := int(query.Period / query.BucketDuration)

	seriesPoints, err := s.walStreamSampleRepository.GetSeries(physical_repositories.WalStreamSeriesQuery{
		DatabaseID:     query.DatabaseID,
		FirstBucketAt:  seriesFrom,
		BucketDuration: query.BucketDuration,
		BucketCount:    bucketCount,
		SampleInterval: wal_health.SampleInterval,
	})
	if err != nil {
		return nil, err
	}

	statistics := &WalStreamStatistics{
		Verdict: wal_health.GetVerdict(wal_health.VerdictInput{
			NewestSample:           newestSample,
			RecentSamples:          recentSamples,
			Streamer:               streamer,
			SessionStartsLastHour:  sessionStartsLastHour,
			SlotRebuildLagBytes:    query.SlotRebuildLagBytes,
			IsStreamingExpected:    query.IsStreamingExpected,
			HasCompletedFullBackup: query.HasCompletedFullBackup,
			NewestChainGaps:        query.NewestChainGaps,
			Now:                    query.Now,
		}),
		KeepingUp:        wal_health.GetKeepingUp(recentSamples, query.SlotRebuildLagBytes, query.Now),
		StreamerState:    wal_health.GetStreamerState(streamer, query.Now),
		RestartsLastHour: wal_health.GetRestartsLastHour(recentSamples, sessionStartsLastHour, query.Now),
		Streamer:         streamer,
		NewestSample:     newestSample,
		SeriesFrom:       seriesFrom,
		SeriesTo:         seriesFrom.Add(time.Duration(bucketCount) * query.BucketDuration),
		SeriesPoints:     seriesPoints,
	}

	if newestCommittedSegment != nil {
		statistics.LastArchivedWalAt = new(newestCommittedSegment.ReceivedAt)
	}

	return statistics, nil
}
