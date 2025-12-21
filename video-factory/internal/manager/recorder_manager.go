package manager

import (
	"context"
	"video-factory/internal/recorder"

	"github.com/rs/zerolog/log"
)

func (m *Manager) StartRecorder() {
	log.Info().Int64("id", m.Id).Str("url", m.CurrentURL).Msg("[Recoder Manager] 启动新录制任务")

	// 创建新 Recorder
	rec, err := recorder.NewRecorder(m.Config, m.CurrentURL, m.Room, m.Streamer.GetOpenTime()/1000)
	if err != nil {
		log.Err(err).Int64("id", m.Id).Str("anchor", m.Room.AnchorName).Msg("[Recoder Manager] 初始化录制器失败")
		return
	}
	m.mu.Lock()
	recordCtx, cancel := context.WithCancel(m.ctx)
	m.recordCancel = cancel
	m.Recorder = rec

	go func() {
		if err := rec.Start(recordCtx); err != nil {
			log.Err(err).Int64("id", m.Id).Str("anchor", m.Room.AnchorName).
				Msg("[Recoder Manager] 录制任务异常退出")
			// 进阶：如果录制频繁失败，是否要触发 Manager 重新刷新 URL？
			log.Warn().Int64("id", m.Id).Str("anchor", m.Room.AnchorName).
				Msg("[Recoder Manager] 录制任务异常，触发刷新")
			m.TriggerRefresh()
		}
	}()
	m.mu.Unlock()
}

func (m *Manager) updateRecorder(streamURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 如果 Recorder 正在运行
	if m.Recorder != nil && m.Recorder.IsRunning() {
		// URL 没变，直接返回
		if m.Recorder.GetSafeURL() == streamURL {
			log.Info().Int64("id", m.Id).Str("anchor", m.Room.AnchorName).
				Msg("[Recoder Manager] 录制URL未变化，不更新")
			return
		}
		// 如果 URL 变了，更新 URL
		log.Info().Int64("id", m.Id).Str("anchor", m.Room.AnchorName).Msg("[Recoder Manager] 更新录制URL")
		m.Recorder.UpdateURL(streamURL)
		return
	}

	// 清理旧引用
	if m.Recorder != nil {
		log.Warn().Int64("id", m.Id).Str("anchor", m.Room.AnchorName).
			Msg("[Manager] 发现已停止的录制器实例，准备重启")
		m.Recorder = nil
		m.recordCancel = nil
	}

	// 启动新的 recorder
	go m.StartRecorder()
}

func (m *Manager) StopRecorder() {
	m.mu.Lock()
	if m.recordCancel != nil {
		log.Info().Int64("id", m.Id).Str("anchor", m.Room.AnchorName).Msg("[Recoder Manager] 停止录制任务")
		m.recordCancel() // 这会触发 Recorder.Start 中的 ctx.Done()
		m.recordCancel = nil
		m.Recorder = nil
	}
	m.mu.Unlock()
}
