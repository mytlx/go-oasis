package manager

import (
	"context"
	"video-factory/internal/recorder"

	"github.com/rs/zerolog/log"
)

func (m *Manager) StartRecorder() {
	log.Info().Int64("id", m.Id).Str("url", m.CurrentURL).Msg("[Manager] 启动新录制任务")

	// 创建新 Recorder
	rec, err := recorder.NewRecorder(m.Config, m.CurrentURL, m.Room, m.Streamer.GetOpenTime()/1000)
	if err != nil {
		log.Err(err).Msg("初始化录制器失败")
		return
	}
	m.mu.Lock()
	recordCtx, cancel := context.WithCancel(m.ctx)
	m.recordCancel = cancel
	m.Recorder = rec

	go func() {
		if err := rec.Start(recordCtx); err != nil {
			log.Err(err).Int64("id", m.Id).Msg("录制任务异常退出")
			// 进阶：如果录制频繁失败，是否要触发 Manager 重新刷新 URL？
			log.Warn().Int64("id", m.Id).Msg("录制任务异常，触发刷新")
			m.TriggerRefresh()
		}
	}()
	m.mu.Unlock()
}

func (m *Manager) updateRecorder(streamURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 如果 Recorder 正在运行且 URL 没变，直接返回
	if m.Recorder != nil && m.Recorder.GetSafeURL() == streamURL {
		return
	}

	// 如果 URL 变了，更新 URL
	if m.Recorder != nil {
		log.Info().Int64("id", m.Id).Msg("[Manager] 更新录制URL")
		m.Recorder.UpdateURL(streamURL)
		return
	}

	// 启动新的 recorder
	go m.StartRecorder()
}

func (m *Manager) StopRecorder() {
	m.mu.Lock()
	if m.recordCancel != nil {
		log.Info().Int64("id", m.Id).Msg("[Manager] 停止录制任务")
		m.recordCancel() // 这会触发 Recorder.Start 中的 ctx.Done()
		m.recordCancel = nil
	}
	m.mu.Unlock()
}
