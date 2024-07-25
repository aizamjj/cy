(def- prefix "ctrl+a")

(def- actions @[])

(defn
  param/rset
  ```Set the value of a parameter at `:root`.```
  [key value]
  (param/set :root key value))

(defmacro key/action
  ````Register an action. Equivalent to the Janet built-in `(defn`), but requires a docstring.

An action is just a Janet function that is registered to the cy server with a short human-readable string description. It provides a convenient method for making some functionality you use often more discoverable.

In a similar way to other modern applications, cy has a command palette (invoked by default with {{bind :root ctrl+a ctrl+p}}, see [`(action/command-palette)
`](/api.md#actioncommand-palette)) in which all registered actions will appear.
````
  [name docstring & body]
  ~(upscope
     (defn ,name ,docstring [] ,;body)
     (,array/push actions [,docstring ,name])))

(defmacro key/bind-many
  ````Bind many bindings at once in the same scope.

For example:
```janet
(key/bind-many :root
               [prefix "j"] action/new-shell
               [prefix "n"] action/new-project)
```
````
  [scope & body]

  (when (not (= (% (length body) 2) 0))
    (error "key/bind-many requires an even number of arguments"))

  (as?-> body _
         (length _)
         (range 0 _ 2)
         (map |(tuple ;(array/slice body $ (+ $ 2))) _)
         (map |(do
                 (def [binding func] $)
                 (tuple 'key/bind scope binding func)) _)))

(defmacro key/bind-many-tag
  ````Bind many bindings at once in the same scope, adding the provided tag.
````
  [scope tag & body]

  (when (not (= (% (length body) 2) 0))
    (error "key/bind-many-tag requires an even number of arguments"))

  (as?-> body _
         (length _)
         (range 0 _ 2)
         (map |(tuple ;(array/slice body $ (+ $ 2))) _)
         (map |(do
                 (def [binding func] $)
                 (tuple 'key/bind scope binding func :tag tag)) _)))

(key/action
  action/command-palette
  "Open the command palette."
  (def binds (key/current))
  (def bound-actions
    (map
      |(do
         (def [desc func] $)
         (def sequence (as?-> binds x
                              (find |(= func ($ :function)) x)
                              (get x :sequence)
                              (string/join x " ")
                              (string " " x " ")))
         (tuple [desc (string sequence)] func))
      actions))

  (as?-> bound-actions _
         (input/find _
                     :full true
                     :reverse true
                     :prompt "search: actions")
         (apply _)))

(defn
  shell/new
  ```Create a new shell initialized in the working directory `path`.```
  [&opt path]
  (default path "")
  (def shells (group/mkdir :root "/shells"))
  (cmd/new shells :path path))

(defn
  shell/attach
  ```Create a new shell initialized in the working directory `path` and attach to it.```
  [&opt path]
  (default path "")
  (def shells (group/mkdir :root "/shells"))
  (pane/attach (cmd/new shells :path path)))

(defn
  layout/type?
  ```Report whether node is of the provided type.```
  [type node]
  (= (node :type) type))

(defn
  layout/pane?
  ```Report whether node is of type :pane.```
  [node]
  (layout/type? :pane node))

(defn
  layout/find-path
  ```Get the path to the first node satisfying the predicate function or nil if none exists.```
  [node predicate]
  (if (predicate node) (break @[]))

  (cond
    (layout/type? :split node) (do
                                 (def {:a a :b b} node)
                                 (def a-path (layout/find-path a predicate))
                                 (if (not (nil? a-path)) (break @[:a ;a-path]))
                                 (def b-path (layout/find-path b predicate))
                                 (if (not (nil? b-path)) (break @[:b ;b-path]))
                                 nil)
    (layout/type? :margins node) (do
                                   (def {:node node} node)
                                   (def path (layout/find-path node predicate))
                                   (if (not (nil? path)) (break @[:node ;path])))
    nil))

(defn
  layout/has?
  ```Report whether this node or one of its descendants matches the predicate function.```
  [node predicate]
  (not (nil? (layout/find-path node predicate))))

(defn
  layout/attached?
  ```Report whether node or one of its descendants is attached.```
  [node]
  (not (nil? (layout/find-path node |($ :attached)))))

(defn
  layout/attach-path
  ```Get the path to the attached node for the given node.```
  [node]
  (layout/find-path node |($ :attached)))

(defn
  layout/path
  ```Resolve the path to a node. Returns nil if any portion of the path is invalid.```
  [node path]
  (if (= (length path) 0) (break node))
  (def [head & rest] path)
  (if (nil? (node head)) (break nil))
  (layout/path (node head) rest))

(defn
  layout/assoc
  ```Set the node at the given path in layout to the provided node. Returns a copy of the original layout with the node changed.```
  [layout path node]
  (if (= (length path) 0) (break node))
  (def [head & rest] path)
  (if (nil? (layout head)) (break layout))
  (def new-layout (struct/to-table layout))
  (put new-layout head (layout/assoc (layout head) rest node))
  (table/to-struct new-layout))

(defn
  layout/get-last
  ```Get the path to the last node in the path where (predicate node) evaluates to true.```
  [layout path predicate]
  # Must be a valid path and actually map to a node
  (if (= (length path) 0) (break nil))
  (if (nil? (layout/path layout path)) (break nil))

  (def found-path
    (find
      |(predicate (layout/path layout (array/slice path ;$)))
      (->>
        (range (length path))
        (map |(tuple 0 $))
        (reverse))))

  (if (nil? found-path) (break nil))
  (array/slice path ;found-path))

(defn
  layout/replace
  ```Replace the node at the path by passing it through a replacer function.```
  [node path replacer]
  (layout/assoc node path (replacer (layout/path node path))))

(defn
  layout/replace-attached
  ```Replace the attached pane in this tree with a new one using the provided replacer function. This function will be invoked with a single argument, the node that is currently attached, and it should return a new node.```
  [node replacer]
  (if (layout/pane? node) (break (replacer node)))
  (def path (layout/attach-path node))
  (if (nil? path) (break node))
  (def current (layout/path node path))
  (layout/assoc node path (replacer current)))

(defn
  layout/detach
  ```Detach the attached node in the tree.```
  [node]
  (layout/replace-attached node |(do {:type :pane :id ($ :id)})))

(defn
  layout/attach
  ```Attach to the node at path in layout.```
  [layout path]
  (layout/replace
    (layout/detach layout)
    path
    |{:type :pane :id ($ :id) :attached true}))

(defn
  layout/find-bottom
  ```Find the path to the node at the bottom.```
  [node]
  (if (layout/pane? node) (break @[]))

  (cond
    (layout/type? :split node) (do
                                 (def {:vertical vertical :a a :b b} node)
                                 (if vertical
                                   @[:b ;(layout/find-bottom b)]
                                   @[:a ;(layout/find-bottom a)]))
    (layout/type? :margins node) @[:node ;(layout/find-bottom (node :node))]
    nil))

(defn
  layout/move
  ```This function attaches to the pane nearest to the one the user is currently attached to along an axis. It returns a new copy of layout with the attachment point changed or returns the same layout if no motion could be completed.

is-axis is a unary function that, given a node, returns a boolean that indicates whether the node is arranged _along the axis in question._ For example, when moving vertically, a vertical split (two panes on top of each other) would return true.

successors is a unary function that, given a node, returns the paths of all of the child nodes accessible from the node in the order of their appearance along the axis. For example, when moving vertically upwards, for a vertical split node this function would return @[[:b] [:a]], because :b is the first node from the bottom, and when moving vertically downwards it would return @[[:a] [:b]] because :a is the first node from the top.
  ```
  [layout is-axis successors]
  (def path (layout/attach-path layout))
  (if (nil? path) (break layout))

  # We look for a path we can attach to in the opposite direction of
  # movement.
  # 
  # Consider the case where a node has successors :a, :b:, and :c arranged
  # along the axis of motion; if we're attached to a node on :b and moving in
  # the direction of :a, we want `detached-successors` to return just [:a],
  # since [:c] is "after" or "below" us.
  (defn detached-successors [node]
    (->>
      (successors node)
      (reverse)
      (take-while |(not (layout/attached? (layout/path node $))))
      (reverse)))

  (defn check-node [node]
    (and (is-axis node) (> (length (detached-successors node)) 0)))

  # We first find the most recent ancestor to the node we're attached to that
  # has a child tree that we can move to.
  (def branch-path (layout/get-last layout path check-node))
  (if (nil? branch-path) (break layout))

  (def [next-path] (detached-successors (layout/path layout branch-path)))
  (def full-path @[;branch-path ;next-path])

  # Find the closest pane we can attach to in the direction of motion.
  (defn
    find-nearest
    [node]
    (if (layout/pane? node) (break @[]))
    (def [nearest] (successors node))
    @[;nearest ;(find-nearest (layout/path node nearest))])

  (layout/attach
    layout
    @[;full-path ;(find-nearest (layout/path layout full-path))]))

(defn
  layout/move-up
  ```Change the layout by moving to the next node "above" the attached pane.```
  [layout]
  (layout/move
    layout
    |(and (layout/type? :split $) ($ :vertical))
    |(cond
       (layout/type? :split $) (if ($ :vertical)
                                 @[[:b] [:a]]
                                 @[[:a] [:b]])
       (layout/type? :margins $) @[[:node]]
       @[[]])))

(defn
  layout/move-down
  ```Change the layout by moving to the next node "below" the attached pane.```
  [layout]
  (layout/move
    layout
    |(and (layout/type? :split $) ($ :vertical))
    |(cond
       (layout/type? :split $) @[[:a] [:b]]
       (layout/type? :margins $) @[[:node]]
       @[[]])))

(defn
  layout/split-right
  ```Split the currently attached pane into two horizontally, replacing the right pane with the given node.```
  [layout node]
  (layout/replace-attached layout |(do {:type :split
                                        :percent 50
                                        :a (layout/detach $)
                                        :b node})))

(defn
  layout/split-left
  ```Split the currently attached pane into two horizontally, replacing the left pane with the given node.```
  [layout node]
  (layout/replace-attached layout |(do {:type :split
                                        :percent 50
                                        :b (layout/detach $)
                                        :a node})))

(defn
  layout/split-down
  ```Split the currently attached pane into two vertically, replacing the bottom pane with the given node.```
  [layout node]
  (layout/replace-attached layout |(do {:type :split
                                        :percent 50
                                        :vertical true
                                        :a (layout/detach $)
                                        :b node})))

(defn
  layout/split-up
  ```Split the currently attached pane into two vertically, replacing the top pane with the given node.```
  [layout node]
  (layout/replace-attached layout |(do {:type :split
                                        :percent 50
                                        :vertical true
                                        :b (layout/detach $)
                                        :a node})))

# TODO(cfoust): 07/25/24 clean this up
(key/action
  action/split-right
  "Split the current pane to the right."
  (def path (cmd/path (pane/current)))
  (def shells (group/mkdir :root "/shells"))
  (def shell (cmd/new shells :path path :name (path/base path)))
  (layout/set
    (layout/split-right
      (layout/get)
      {:type :pane :id shell :attached true})))

(key/action
  action/split-right
  "Split the current pane to the left"
  (def path (cmd/path (pane/current)))
  (def shells (group/mkdir :root "/shells"))
  (def shell (cmd/new shells :path path :name (path/base path)))
  (layout/set
    (layout/split-left
      (layout/get)
      {:type :pane :id shell :attached true})))

(key/action
  action/split-down
  "Split the current pane downwards."
  (def path (cmd/path (pane/current)))
  (def shells (group/mkdir :root "/shells"))
  (def shell (cmd/new shells :path path :name (path/base path)))
  (layout/set
    (layout/split-down
      (layout/get)
      {:type :pane :id shell :attached true})))

(key/action
  action/split-up
  "Split the current pane upwards"
  (def path (cmd/path (pane/current)))
  (def shells (group/mkdir :root "/shells"))
  (def shell (cmd/new shells :path path :name (path/base path)))
  (layout/set
    (layout/split-up
      (layout/get)
      {:type :pane :id shell :attached true})))

(key/action
  action/move-up
  "Move up to the next pane."
  (layout/set (layout/move-up (layout/get))))

(key/action
  action/move-down
  "Move down to the next pane."
  (layout/set (layout/move-down (layout/get))))

(key/action
  action/new-shell
  "Create a new shell."
  (def path (cmd/path (pane/current)))
  (def shells (group/mkdir :root "/shells"))
  (def shell (cmd/new shells :path path :name (path/base path)))
  (pane/attach shell))

(key/action
  action/new-project
  "Create a new project."
  (def path (cmd/path (pane/current)))
  (def projects (group/mkdir :root "/projects"))
  (def project (group/new projects :name (path/base path)))
  (def editor
    (cmd/new project
             :path path
             :name "editor"
             :command (os/getenv "EDITOR" "vim")))
  (def shell (cmd/new project :path path :name "shell"))
  (pane/attach editor))

(key/action
  action/jump-project
  "Jump to a project."
  (def projects (group/mkdir :root "/projects"))
  (as?-> projects _
         (group/children _)
         (map
           |(tuple
              (tree/name $)
              {:type :node
               :id ((group/children $) 0)}
              $)
           _)
         (input/find _ :prompt "search: project")
         (group/children _)
         (_ 0) # Gets the first index, the editor
         (pane/attach _)))

(key/action
  action/jump-shell
  "Jump to a shell."
  (def shells (group/mkdir :root "/shells"))
  (as?-> (group/children shells) _
         (map |(tuple
                 (cmd/path $)
                 {:type :node
                  :id $}
                 $) _)
         (input/find _ :prompt "search: shell")
         (pane/attach _)))

(key/action
  action/next-pane
  "Move to the next pane."
  (def children
    (-?>>
      (pane/current)
      (tree/parent)
      (group/children)
      (filter tree/pane?)))

  (when (nil? children) (break))

  (def index (index-of (pane/current) children))

  (def next-panes
    (array/concat
      (array)
      (array/slice children (+ index 1))
      (array/slice children 0 index)))

  (when (= 0 (length next-panes)) (break))
  (def [next] next-panes)
  (pane/attach next))

(key/action
  action/rename-pane
  "Rename the current pane."
  (def pane (pane/current))
  (def old-path (tree/path pane))
  (as?-> pane _
         (input/text (string "rename: " (tree/path pane))
                     :preset (tree/name pane))
         (do (tree/set-name pane _) _)
         (msg/toast :info (string "renamed " old-path " to " (tree/path pane)))))

(key/action
  action/jump-pane
  "Jump to a pane."
  (as?-> (group/leaves :root) _
         (map |(tuple (tree/path $) {:type :node :id $} $) _)
         (input/find _ :prompt "search: pane")
         (pane/attach _)))

(key/action
  action/jump-screen-lines
  "Jump to a pane based on screen lines."
  (as?-> (group/leaves :root) _
         (mapcat
           (fn [id]
             (->> id
                  (pane/screen)
                  (filter (complement |(string/check-set " " $)))
                  (map
                    |(tuple [$ (tree/path id)] {:type :node :id id} id))))
           _)
         (input/find _ :prompt "search: screen")
         (pane/attach _)))

(key/action
  action/kill-current-pane
  "Kill the current pane."
  (tree/kill (pane/current)))

(key/action
  action/kill-server
  "Kill the cy server."
  (cy/kill-server))

(key/action
  action/detach
  "Detach from the cy server."
  (cy/detach))

#(key/action
#action/toggle-margins
#"Toggle the screen's margins."
#(def size (viewport/size))
#(case (+ (size 0) (size 1))
#0 (viewport/set-size [0 80])
#(viewport/set-size [0 0])))

#(key/action
#action/margins-80
#"Set size to 80 columns."
#(viewport/set-size [0 80]))

#(key/action
#action/margins-160
#"Set size to 160 columns."
#(viewport/set-size [0 160]))

(key/action
  action/choose-frame
  "Choose a frame."
  (as?-> (viewport/get-frames) _
         (map |(tuple $ {:type :frame :name $} $) _)
         (input/find _ :prompt "search: frame")
         (viewport/set-frame _)))

(key/action
  action/browse-animations
  "Browse animations."
  (as?-> (viewport/get-animations) _
         (map |(tuple $ {:type :animation :name $} $) _)
         (input/find _
                     # so we don't confuse the user
                     :animated false
                     :prompt "search: animation")))

(key/action
  action/reload-config
  "Reload the cy configuration."
  (cy/reload-config))

(key/action
  action/random-frame
  "Switch to a random frame."
  (def frames (viewport/get-frames))
  (def rng (math/rng))
  (viewport/set-frame (get frames (math/rng-int rng (length frames)))))

#(key/action
#action/margins-smaller
#"Decrease margins by 5 columns."
#(def [lines cols] (viewport/size))
#(viewport/set-size [lines (+ cols 10)]))

#(key/action
#action/margins-bigger
#"Increase margins by 5 columns."
#(def [lines cols] (viewport/size))
#(viewport/set-size [lines (- cols 10)]))

(key/action
  action/open-log
  "Open a .borg file."
  (as?-> (path/glob (path/join [(param/get :data-directory) "*.borg"])) _
         (map |(tuple $ {:type :replay :path $} $) _)
         (input/find _ :prompt "search: log file")
         (replay/open-file :root _)
         (pane/attach _)))

(defn- get-pane-commands [id result-func]
  (var [ok commands] (protect (cmd/commands id)))
  (if (not ok) (set commands @[]))
  (default commands @[])
  (map |(tuple [(string/replace-all "\n" "↵" ($ :text)) (tree/path id)]
               {:type :scrollback
                :focus ((($ :input) 0) :from)
                :highlights @[(($ :input) 0)]
                :id id}
               (result-func $)) commands))

(key/action
  action/jump-pane-command
  "Jump to a pane based on a command."
  (as?-> (group/leaves :root) _
         (mapcat |(get-pane-commands $ (fn [cmd] $)) _)
         (input/find _ :prompt "search: pane (command)")
         (pane/attach _)))

(key/action
  action/jump-command
  "Jump to the output of a command."
  (as?-> (group/leaves :root) _
         (mapcat |(get-pane-commands $ (fn [cmd] [$ cmd])) _)
         (input/find _ :prompt "search: command")
         (let [[id cmd] _]
           (pane/attach id)
           (replay/open
             id
             :main true
             :location (((cmd :input) 0) :from)))))

(key/action
  action/open-replay
  "Enter replay mode for the current pane."
  (replay/open (pane/current)))

(key/bind-many-tag :root "general"
                   [prefix "ctrl+p"] action/command-palette
                   [prefix "q"] action/kill-server
                   [prefix "d"] action/detach
                   [prefix "F"] action/choose-frame
                   [prefix "p"] action/open-replay
                   [prefix "r"] action/reload-config
                   [prefix "P"] cy/paste)

(key/bind-many-tag :root "panes"
                   [prefix "ctrl+i"] pane/history-forward
                   [prefix "ctrl+o"] pane/history-backward
                   [prefix "x"] action/kill-current-pane
                   [prefix "C"] action/jump-command
                   [prefix ":"] action/jump-screen-lines
                   [prefix "j"] action/new-shell
                   [prefix "n"] action/new-project
                   [prefix "k"] action/jump-project
                   [prefix "l"] action/jump-shell
                   [prefix ";"] action/jump-pane
                   [prefix "c"] action/jump-pane-command)

#(key/bind-many-tag :root "viewport"
#[prefix "g"] action/toggle-margins
#[prefix "1"] action/margins-80
#[prefix "2"] action/margins-160
#[prefix "+"] action/margins-smaller
#[prefix "-"] action/margins-bigger)

(key/bind-many-tag :root "unprefixed"
                   ["ctrl+l"] action/next-pane)

(key/action
  action/replay-playback-1x
  "Set the playback rate to 1x real time."
  (replay/time-playback-rate 1))

(key/action
  action/replay-playback-2x
  "Set the playback rate to 2x real time."
  (replay/time-playback-rate 2))

(key/action
  action/replay-playback-5x
  "Set the playback rate to 5x real time."
  (replay/time-playback-rate 5))

(key/action
  action/replay-playback-reverse-1x
  "Set the playback rate to -1x real time (backwards)."
  (replay/time-playback-rate -1))

(key/action
  action/replay-playback-reverse-2x
  "Set the playback rate to -2x real time (backwards)."
  (replay/time-playback-rate -2))

(key/action
  action/replay-playback-reverse-5x
  "Set the playback rate to -5x real time (backwards)."
  (replay/time-playback-rate -5))

(key/bind-many-tag :time "general"
                   ["q"] replay/quit
                   ["ctrl+c"] replay/quit
                   ["esc"] replay/quit
                   ["]" "c"] replay/command-forward
                   ["[" "c"] replay/command-backward
                   ["right"] replay/time-step-forward
                   ["left"] replay/time-step-back
                   ["/"] replay/search-forward
                   ["?"] replay/search-backward
                   ["g" "g"] replay/beginning
                   ["n"] replay/search-again
                   ["N"] replay/search-reverse
                   [" "] replay/time-play
                   ["1"] action/replay-playback-1x
                   ["2"] action/replay-playback-2x
                   ["3"] action/replay-playback-5x
                   ["!"] action/replay-playback-reverse-1x
                   ["@"] action/replay-playback-reverse-2x
                   ["#"] action/replay-playback-reverse-5x
                   ["G"] replay/end)

(key/bind-many-tag :copy "general"
                   ["v"] replay/select
                   ["y"] replay/copy)

(key/bind-many-tag :copy "motion"
                   ["g" "g"] replay/beginning
                   ["G"] replay/end
                   ["/"] replay/search-forward
                   ["?"] replay/search-backward
                   ["q"] replay/quit
                   ["ctrl+c"] replay/quit
                   ["esc"] replay/quit
                   ["left"] replay/cursor-left
                   ["l"] replay/cursor-right
                   # ??? <BS> in vim actually goes across lines
                   ["backspace"] replay/cursor-left
                   ["right"] replay/cursor-right
                   # ??? <space> in vim actually goes across lines
                   [" "] replay/cursor-right
                   ["h"] replay/cursor-left
                   ["ctrl+h"] replay/cursor-left
                   ["ctrl+u"] replay/half-page-up
                   ["ctrl+d"] replay/half-page-down
                   ["up"] replay/scroll-up
                   ["down"] replay/scroll-down
                   ["j"] replay/cursor-down
                   ["k"] replay/cursor-up
                   ["n"] replay/search-again
                   ["N"] replay/search-reverse
                   ["s"] replay/swap-screen
                   ["w"] replay/word-forward
                   ["b"] replay/word-backward
                   ["e"] replay/word-end-forward
                   ["g" "e"] replay/word-end-backward
                   ["W"] replay/big-word-forward
                   ["B"] replay/big-word-backward
                   ["E"] replay/big-word-end-forward
                   ["g" "E"] replay/big-word-end-backward
                   ["]" "c"] replay/command-forward
                   ["[" "c"] replay/command-backward
                   ["]" "C"] replay/command-select-forward
                   ["[" "C"] replay/command-select-backward
                   ["0"] replay/start-of-line
                   ["home"] replay/start-of-line
                   ["g" "M"] replay/middle-of-line
                   ["$"] replay/end-of-line
                   ["^"] replay/first-non-blank
                   ["g" "_"] replay/last-non-blank
                   ["g" "0"] replay/start-of-screen-line
                   ["g" "home"] replay/start-of-screen-line
                   ["g" "m"] replay/middle-of-screen-line
                   ["g" "$"] replay/end-of-screen-line
                   ["g" "^"] replay/first-non-blank-screen
                   ["g" "end"] replay/last-non-blank-screen
                   [";"] replay/jump-again
                   [","] replay/jump-reverse
                   ["f" [:re "."]] replay/jump-forward
                   ["F" [:re "."]] replay/jump-backward
                   ["t" [:re "."]] replay/jump-to-forward
                   ["T" [:re "."]] replay/jump-to-backward)
