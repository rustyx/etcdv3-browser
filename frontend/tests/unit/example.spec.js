import { expect } from 'chai'
import { shallowMount } from '@vue/test-utils'
import Browser from '@/components/Browser.vue'

describe('Browser.vue', () => {
  it('renders', () => {
    const wrapper = shallowMount(Browser, {
    })
    expect(wrapper.text()).to.include('Select')
  })
})
