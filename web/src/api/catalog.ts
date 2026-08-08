import { apiClient } from './client'
import type { ApiEnvelope } from '@/types/api'
import type { Campaign, Channel, Creative, Game } from '@/types/catalog'

export type GameInput = Omit<Game, 'id'>
export type ChannelInput = Omit<Channel, 'id'>
export type CampaignInput = Omit<Campaign, 'id'>
export type CreativeInput = Omit<Creative, 'id'>

async function list<T>(path: string): Promise<T[]> { const { data } = await apiClient.get<ApiEnvelope<T[]>>(path); return data.data }
async function create<T, P>(path: string, payload: P): Promise<T> { const { data } = await apiClient.post<ApiEnvelope<T>>(path, payload); return data.data }
async function update<T, P>(path: string, payload: P): Promise<T> { const { data } = await apiClient.put<ApiEnvelope<T>>(path, payload); return data.data }

export const catalogApi = {
  listGames: () => list<Game>('/games'),
  createGame: (payload: GameInput) => create<Game, GameInput>('/games', payload),
  updateGame: (id: string, payload: GameInput) => update<Game, GameInput>(`/games/${id}`, payload),
  listChannels: () => list<Channel>('/channels'),
  createChannel: (payload: ChannelInput) => create<Channel, ChannelInput>('/channels', payload),
  updateChannel: (id: string, payload: ChannelInput) => update<Channel, ChannelInput>(`/channels/${id}`, payload),
  listCampaigns: () => list<Campaign>('/campaigns'),
  createCampaign: (payload: CampaignInput) => create<Campaign, CampaignInput>('/campaigns', payload),
  updateCampaign: (id: string, payload: CampaignInput) => update<Campaign, CampaignInput>(`/campaigns/${id}`, payload),
  listCreatives: () => list<Creative>('/creatives'),
  createCreative: (payload: CreativeInput) => create<Creative, CreativeInput>('/creatives', payload),
  updateCreative: (id: string, payload: CreativeInput) => update<Creative, CreativeInput>(`/creatives/${id}`, payload),
}
