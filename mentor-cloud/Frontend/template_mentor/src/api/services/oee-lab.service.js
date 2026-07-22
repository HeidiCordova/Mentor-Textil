import client from '../client'

export const oeeLabService = {
  list()          { return client.get('/oee-lab') },
  get(id)         { return client.get(`/oee-lab/${id}`) },
  save(body)      { return client.post('/oee-lab', body) },
  remove(id)      { return client.delete(`/oee-lab/${id}`) },
}
