// import Vue from 'vue'
import Vuetify from 'vuetify'
import { mount, createLocalVue } from '@vue/test-utils'
import { expect } from 'chai'
import TreeItem from '@/components/TreeItem.vue'

const localVue = createLocalVue()

describe('TreeItem.vue', () => {
  let vuetify
  beforeEach(() => {
    vuetify = new Vuetify()
  })
  it('renders', () => {
    const wrapper = mount(TreeItem, {
      localVue,
      vuetify,
      propsData: {
        item: { isRoot: true },
      },
    })
    expect(wrapper.html()).to.include('item')
  })
})
