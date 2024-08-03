(test ":pane"
      (layout/set {:type :pane :attached true :id 2})
      (layout/set {:type :pane :id 2 :attached true})
      (layout/set {:type :pane :attached true}))

(test ":pane 2"
      (layout/set {:type :pane :attached true}))

(test "layout/get"
      (def layout {:type :pane
                   :id 2
                   :remove-on-exit false
                   :attached true})
      (layout/set layout)
      (assert (deep= (layout/get) layout)))

(test "borders"
      (do
        (def layout {:type :split
                     :percent 26
                     :border :normal
                     :vertical false
                     :a {:type :pane :attached true}
                     :b {:type :pane :attached false}})
        (layout/set layout)
        (assert (deep= (layout/get) layout)))

      (do
        (def layout {:type :split
                     :percent 26
                     :border :none
                     :vertical false
                     :a {:type :pane :attached true}
                     :b {:type :pane :attached false}})
        (layout/set layout)
        (assert (deep= (layout/get) layout)))

      (expect-error (layout/set
                      {:type :split
                       :percent 26
                       :border :asd
                       :vertical false
                       :a {:type :pane :attached true}
                       :b {:type :pane :attached false}})))

(test ":split"
      (layout/set
        {:type :split
         # horizontal
         :percent 26
         :a {:type :pane :attached true}
         :b {:type :pane}})

      (layout/set
        {:type :split
         :vertical true
         :percent 26
         :a {:type :pane :attached true}
         :b {:type :pane}})

      (expect-error (layout/set
                      {:type :split
                       :vertical true
                       # don't let both of these get set
                       :cells 20
                       :percent 26
                       :a {:type :pane :attached true}
                       :b {:type :pane}})))
(test ":margins"
      (layout/set
        {:type :margins
         :cols 20
         :rows 0
         :node {:type :pane :attached true}})

      # omit a bunch of fields
      (layout/set
        {:type :margins
         :node {:type :pane :attached true}}))

(test ":border"
      (layout/set
        {:type :borders
         :title "test"
         :border :double
         :node {:type :pane :attached true}})

      (layout/set
        {:type :borders
         :node {:type :pane :attached true}}))

(test "bad input"
      # bad node
      (expect-error (layout/set
                      {:type :margins
                       :cols 20
                       :rows 0
                       :node 1213132}))

      (expect-error (layout/set
                      {:type :split
                       :vertical true
                       :cells 20
                       :percent 26
                       :a {:type :pane :attached true}
                       :b {:type :pane :attached true}})))


(test "layout/attach-path"
      (assert (deep=
                (layout/attach-path
                  {:type :margins
                   :cols 20
                   :rows 0
                   :node {:type :pane :attached true}})
                @[:node]))

      (assert (deep=
                (layout/attach-path
                  {:type :split
                   :percent 50
                   :a {:type :pane}
                   :b {:type :pane :attached true}})
                @[:b]))

      (assert (deep=
                (layout/attach-path
                  {:type :margins
                   :cols 20
                   :rows 0
                   :node {:type :split
                          :vertical true
                          :cells 20
                          :percent 26
                          :a {:type :pane :attached true}
                          :b {:type :pane}}})
                @[:node :a])))


(test "layout/path"
      (assert (deep=
                (layout/path
                  {:type :margins
                   :cols 20
                   :rows 0
                   :node {:type :pane :attached true}}
                  @[:node])
                {:type :pane :attached true}))

      (assert (deep=
                (layout/path
                  {:type :margins
                   :cols 20
                   :rows 0
                   :node {:type :split
                          :percent 50
                          :a {:type :pane :attached true}
                          :b {:type :pane}}}
                  @[:node :a])
                {:type :pane :attached true}))

      (assert (deep=
                (layout/path
                  {:type :margins
                   :cols 20
                   :rows 0
                   :node {:type :pane :attached true}}
                  @[:a])
                nil)))

(test "layout/find"
      (assert (deep=
                (layout/find
                  {:type :margins
                   :cols 20
                   :rows 0
                   :node {:type :pane :attached true}}
                  |($ :attached))
                @[:node]))

      (assert (deep=
                (layout/find
                  {:type :margins
                   :cols 20
                   :rows 0
                   :node {:type :pane :attached true}}
                  |(layout/type? :margins $))
                @[])))

(test "layout/assoc"
      (assert (deep=
                (layout/assoc
                  {:type :margins
                   :cols 20
                   :rows 0
                   :node {:type :pane :attached true}}
                  @[:node]
                  {:type :pane})
                {:type :margins
                 :cols 20
                 :rows 0
                 :node {:type :pane}}))

      # Does nothing, bad path
      (assert (deep=
                (layout/assoc
                  {:type :margins
                   :cols 20
                   :rows 0
                   :node {:type :pane :attached true}}
                  @[:a]
                  {:type :pane})
                {:type :margins
                 :cols 20
                 :rows 0
                 :node {:type :pane :attached true}})))

(test "layout/detach"
      (assert (deep=
                (layout/detach
                  {:type :margins
                   :cols 20
                   :rows 0
                   :node {:type :pane :attached true}})
                {:type :margins
                 :cols 20
                 :rows 0
                 :node {:type :pane :id nil}})))

(test "layout/split-*"
      (assert (deep=
                (layout/split-right
                  {:type :margins
                   :cols 20
                   :node {:type :pane :attached true}}
                  {:type :pane :id 2 :attached true})

                {:type :margins
                 :cols 20
                 :node {:type :split
                        :percent 50
                        :a {:type :pane}
                        :b {:type :pane :id 2 :attached true}}}))

      (assert (deep=
                (layout/split-down (layout/split-right
                                     {:type :pane :attached true}
                                     {:type :pane :attached true})
                                   {:type :pane :id 2 :attached true})

                {:type :split
                 :percent 50
                 :a {:type :pane}
                 :b {:type :split
                     :vertical true
                     :percent 50
                     :a {:type :pane}
                     :b {:type :pane :id 2 :attached true}}})))


(test "layout/find-last"
      (def layout
        {:type :margins
         :cols 20
         :node {:type :margins
                :cols 20
                :node {:type :pane :attached true}}})

      (def path (layout/attach-path layout))

      (assert (deep=
                (layout/find-last layout path |(layout/type? :margins $))
                @[:node])))

(test "layout/move-up"
      (assert (deep=
                (layout/move-up
                  {:type :split
                   :vertical true
                   :a {:type :pane}
                   :b {:type :margins :node {:type :pane :attached true}}})

                {:type :split
                 :vertical true
                 :a {:type :pane :attached true}
                 :b {:type :margins :node {:type :pane}}}))

      (assert (deep=
                (layout/move-up
                  {:type :split
                   :vertical true
                   :a {:type :split
                       :vertical true
                       :a {:type :pane}
                       :b {:type :pane}}
                   :b {:type :split
                       :vertical true
                       :a {:type :pane :attached true}
                       :b {:type :pane}}})

                {:type :split
                 :vertical true
                 :a {:type :split
                     :vertical true
                     :a {:type :pane}
                     :b {:type :pane :attached true}}
                 :b {:type :split
                     :vertical true
                     :a {:type :pane}
                     :b {:type :pane}}})))


(test "layout/move-down"
      (assert (deep=
                (layout/move-down
                  {:type :split
                   :vertical true
                   :a {:type :pane :attached true}
                   :b {:type :margins :node {:type :pane}}})

                {:type :split
                 :vertical true
                 :a {:type :pane}
                 :b {:type :margins :node {:type :pane :attached true}}})))

(test "layout/move-left"
      (assert (deep=
                (layout/move-left
                  {:type :split
                   :a {:type :pane}
                   :b {:type :borders
                       :node {:type :pane
                              :attached true}}})

                {:type :split
                 :a {:type :pane :attached true}
                 :b {:type :borders
                     :node {:type :pane}}})))

(test "layout/map"
      (assert (deep=
                (layout/map
                  |(if (layout/pane? $) {:type :pane :id 2} $)
                  {:type :split
                   :vertical true
                   :a {:type :pane :attached true}
                   :b {:type :margins :node {:type :pane}}})

                {:type :split
                 :vertical true
                 :a {:type :pane :id 2}
                 :b {:type :margins :node {:type :pane :id 2}}})))

(test "action/add-borders"
      (action/split-right)
      (action/add-borders)
      (action/move-left)
      (action/add-borders)
      (action/add-borders))

(test "action/set-layout-borders"
      (layout/set
        {:type :borders
         :title "test"
         :border :double
         :node {:type :pane :attached true}})

      (action/set-layout-borders))
